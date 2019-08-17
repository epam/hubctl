package api

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"hub/aws"
	"hub/bindata"
	"hub/config"
	"hub/util"
)

type ImportConfig struct {
	TemplateNameFormat string
	SecretsOrder       []Secret
}

var importConfigs = map[string]ImportConfig{
	"k8s-aws": {
		"K8S AWS Adapter in %s",
		[]Secret{
			{"kubernetes.api.clientCert", "certificate", nil},
			{"kubernetes.api.clientKey", "privateKey", nil},
			{"kubernetes.api.caCert", "certificate", nil},
		},
	},
	"metal": {
		"Bare-metal Adapter in %s",
		[]Secret{
			{"kubernetes.api.clientCert", "certificate", nil},
			{"kubernetes.api.clientKey", "privateKey", nil},
			{"kubernetes.api.caCert", "certificate", nil},
		},
	},
	"eks": {
		"EKS Adapter in %s",
		[]Secret{
			{"kubernetes.api.caCert", "certificate", nil}, // discovered by CLI via API
		},
	},
	"gke": {
		"GKE Adapter in %s",
		nil, // discovered by import component
	},
	"aks": {
		"AKS Adapter in %s",
		nil, // discovered by import component
	},
	"openshift": {
		"OpenShift Adapter in %s",
		[]Secret{
			{"kubernetes.api.caCert", "certificate", nil}, // optional
		},
	},
}

func ImportKubernetes(kind, name, environment, template string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	pems io.Reader, clusterBearerToken,
	nativeRegion, nativeEndpoint, nativeClusterName,
	ingressIpOrHost, azureResourceGroup string) {

	err := errors.New("Not implemented")
	if importConfig, exist := importConfigs[kind]; exist {
		err = importK8s(importConfig, kind, name, environment, template,
			autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun,
			pems, clusterBearerToken,
			nativeRegion, nativeEndpoint, nativeClusterName, ingressIpOrHost, azureResourceGroup)
	}
	if err != nil {
		log.Fatalf("Unable to import `%s` Kubernetes: %v", kind, err)
	}
}

func importK8s(importConfig ImportConfig, kind, name, environmentSelector, templateSelector string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	pems io.Reader, clusterBearerToken,
	nativeRegion, nativeEndpoint, nativeClusterName,
	ingressIpOrHost, azureResourceGroup string) error {

	environment, err := environmentBy(environmentSelector)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Environment: %v", err)
	}
	cloudAccount, err := cloudAccountById(environment.CloudAccount)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Cloud Account: %v", err)
	}

	if strings.Contains(name, ".") {
		suffix := "." + cloudAccount.BaseDomain
		i := strings.LastIndex(name, suffix)
		if !strings.HasSuffix(name, suffix) || i < 1 {
			return fmt.Errorf("`%s` looks like FQDN, but Cloud Account base domain is `%s`", name, cloudAccount.BaseDomain)
		}
		name = name[:i]
	}
	fqdn := fmt.Sprintf("%s.%s", name, cloudAccount.BaseDomain)

	ingressIp := ""
	ingressHost := ""

	if kind == "eks" {
		if !strings.HasPrefix(cloudAccount.Kind, "aws") {
			return fmt.Errorf("Cloud Account %s is not AWS but `%s`", cloudAccount.BaseDomain, cloudAccount.Kind)
		}
		if nativeEndpoint == "" {
			cac, err := awsCloudAccountCredentials(cloudAccount.Id)
			if err != nil {
				return err
			}
			regions := make([]string, 0, 2)
			caRegion := cloudAccountRegion(cloudAccount)
			if caRegion != "" {
				if config.Debug {
					log.Printf("Cloud Account `%s` default region is `%s`", cloudAccount.Id, caRegion)
				}
				regions = append(regions, caRegion)
			}
			if config.AwsRegion != "" && !util.Contains(regions, config.AwsRegion) {
				regions = append(regions, config.AwsRegion)
			}
			for _, region := range regions {
				endpoint, ca, err := aws.DescribeEKSClusterWithStaticCredentials(region, nativeClusterName,
					cac.AccessKey, cac.SecretKey, cac.SessionToken)
				if err != nil {
					util.Warn("Unable to retrieve EKS cluster `%s` info in `%s` region: %v",
						nativeClusterName, region, err)
					continue
				}
				if endpoint != "" {
					if config.Verbose {
						log.Printf("Found EKS cluster `%s` in %s region with endpoint %s",
							nativeClusterName, region, endpoint)
					}
					if strings.HasPrefix(endpoint, "https://") && len(endpoint) > 8 {
						endpoint = endpoint[8:]
					}
					nativeEndpoint = endpoint
					if ca != nil && len(ca) > 0 {
						pems = bytes.NewReader(ca)
					}
					break
				}
			}
			if nativeEndpoint == "" {
				log.Fatal("EKS cluster endpoint (--eks-endpoint) must be provided")
			}
		}
	} else if kind == "metal" {
		if ingressIpOrHost == "" {
			parts := strings.Split(nativeEndpoint, ":")
			if len(parts) > 0 {
				ingressIpOrHost = parts[0]
			}
			if ingressIpOrHost == "" {
				log.Fatalf("Cannot determine ingress IP/hostname from API endpoint `%s`", nativeEndpoint)
			}
		}
		if net.ParseIP(ingressIpOrHost) != nil {
			ingressIp = ingressIpOrHost
		} else {
			ingressHost = ingressIpOrHost
		}
	}

	secrets, err := readImportSecrets(importConfig.SecretsOrder, pems)
	if err != nil {
		if !(kind == "openshift" && secrets != nil) { // OpenShift CA is optional
			return fmt.Errorf("Unable to read auth secrets: %v", err)
		}
	}
	hasCaCert := false
	for _, s := range secrets {
		if s.Name == "kubernetes.api.caCert" {
			hasCaCert = true
			break
		}
	}

	adapterTag := "adapter=" + kind
	if templateSelector == "" && autoCreateTemplate {
		templateSelector = fmt.Sprintf(importConfig.TemplateNameFormat, environment.Name)
	}

	var template *StackTemplate

	if templateSelector != "" && !createNewTemplate {
		template, err = templateBy(templateSelector)
		if err != nil && !strings.HasSuffix(err.Error(), " found") { // TODO proper 404 handling
			return fmt.Errorf("Unable to retrieve adapter Template: %v", err)
		}
		if template != nil && (template.Tags == nil || !util.Contains(template.Tags, adapterTag)) {
			util.Warn("Template `%s` [%s] contain no `%s` tag", template.Name, template.Id, adapterTag)
		}
	}

	if template == nil {
		if !autoCreateTemplate {
			return fmt.Errorf("No adapter Template found by `%s`", templateSelector)
		}

		asset := fmt.Sprintf("%s/%s-adapter-template.json.template", requestsBindata, kind)
		templateBytes, err := bindata.Asset(asset)
		if err != nil {
			return fmt.Errorf("No %s embedded: %v", asset, err)
		}
		var templateRequest StackTemplateRequest
		err = json.Unmarshal(templateBytes, &templateRequest)
		if err != nil {
			return fmt.Errorf("Unable to unmarshall JSON into Template request: %v", err)
		}

		// TODO with createNewTemplate = true we can get a 400 HTTP
		// due to duplicate Template name which should be unique across organization
		templateRequest.Name = templateSelector // let use user-supplied selector as Template name, hope it's not id
		templateRequest.Tags = []string{adapterTag}
		templateRequest.TeamsPermissions = environment.TeamsPermissions // copy permissions from Environment

		templateBytes, err = json.Marshal(&templateRequest)
		if err != nil {
			return fmt.Errorf("Unable to marshall Template request into JSON: %v", err)
		}
		template, err = createTemplate(bytes.NewReader(templateBytes))
		if err != nil {
			return fmt.Errorf("Unable to create adapter Template: %v", err)
		}
		err = initTemplate(template.Id)
		if err != nil {
			return fmt.Errorf("Unable to initialize adapter Template: %v", err)
		}
		if config.Verbose {
			log.Printf("Created %s adapter template `%s`", kind, template.Name)
		}
	}

	asset := fmt.Sprintf("%s/%s-adapter-instance.json.template", requestsBindata, kind)
	instanceBytes, err := bindata.Asset(asset)
	if err != nil {
		return fmt.Errorf("No %s embedded: %v", asset, err)
	}
	var instanceRequest StackInstanceRequest
	err = json.Unmarshal(instanceBytes, &instanceRequest)
	if err != nil {
		return fmt.Errorf("Unable to unmarshall JSON into Stack Instance request: %v", err)
	}

	instanceRequest.Name = name
	instanceRequest.Tags = template.Tags
	instanceRequest.Environment = environment.Id
	instanceRequest.Template = template.Id
	parameters := make([]Parameter, 0, len(instanceRequest.Parameters))
	for _, p := range instanceRequest.Parameters {
		rm := false
		switch p.Name {
		case "dns.domain":
			p.Value = fqdn
		case "cloud.region": // TODO cloud.availabilityZone for GKE cluster deployed into single zone
			if nativeRegion != "" {
				p.Value = nativeRegion
			} else {
				rm = true
			}
		case "kubernetes.api.endpoint":
			if nativeEndpoint != "" {
				p.Value = nativeEndpoint
			} else {
				rm = true
			}
		case "kubernetes.api.caCert":
			if !hasCaCert {
				rm = true
			}
		case "kubernetes.eks.cluster", "kubernetes.gke.cluster", "kubernetes.aks.cluster":
			p.Value = nativeClusterName
		case "component.ingress.staticIp":
			p.Value = ingressIp
		case "component.ingress.staticHost":
			p.Value = ingressHost
		case "cloud.azureResourceGroupName":
			if azureResourceGroup != "" {
				p.Value = azureResourceGroup
			} else {
				rm = true
			}
		}
		if !rm {
			parameters = append(parameters, p)
		}
	}
	instanceRequest.Parameters = parameters

	instanceBytes, err = json.Marshal(&instanceRequest)
	if err != nil {
		return fmt.Errorf("Unable to marshall Stack Instance request into JSON: %v", err)
	}
	instance, err := createStackInstance(bytes.NewReader(instanceBytes))
	if err != nil {
		return fmt.Errorf("Unable to create adapter Stack Instance: %v", err)
	}

	if kind == "openshift" && clusterBearerToken != "" {
		secrets = append(secrets,
			Secret{"kubernetes.api.token", "bearerToken", map[string]string{"bearerToken": clusterBearerToken}})
	}
	for _, secret := range secrets {
		id, err := createSecret(stackInstancesResource, instance.Id, secret.Name, "", secret.Kind, secret.Values)
		if err != nil {
			return err
		}
		if config.Verbose {
			log.Printf("Created %s secret with id %s", secret.Name, id)
		}
	}

	err = commandStackInstance(instance.Id, "deploy", waitAndTailDeployLogs, dryRun)
	if err != nil {
		return fmt.Errorf("Unable to deploy adapter Stack Instance: %v", err)
	}

	return nil
}

var pemBlockTypeToSecretKind = map[string]string{
	"CERTIFICATE":     "certificate",
	"RSA PRIVATE KEY": "privateKey",
}

func readImportSecrets(secretsOrder []Secret, pems io.Reader) ([]Secret, error) {
	if secretsOrder == nil {
		return nil, nil
	}
	if config.Verbose {
		stdin := ""
		if pems == os.Stdin {
			stdin = " from stdin"
		}
		log.Printf("Reading TLS auth certs and key%s", stdin)
	}
	pemBytes, err := ioutil.ReadAll(pems)
	if err != nil {
		return nil, err
	}
	var block *pem.Block
	secrets := make([]Secret, 0, len(secretsOrder))
	for _, secret := range secretsOrder {
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}
		if blockKind, exist := pemBlockTypeToSecretKind[block.Type]; !exist || blockKind != secret.Kind {
			return nil, fmt.Errorf("Unexpected PEM block `%s` while reading %s %s ",
				block.Type, secret.Kind, secret.Name)
		}
		secret.Values = make(map[string]string)
		secret.Values[secret.Kind] = string(pem.EncodeToMemory(block))
		secrets = append(secrets, secret)
		if len(pemBytes) == 0 {
			break
		}
	}
	if len(secrets) < len(secretsOrder)-1 {
		err = fmt.Errorf("Expected at least %d secrets, read %d", len(secretsOrder)-1, len(secrets))
	}
	return secrets, err
}
