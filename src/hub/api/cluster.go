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

type ClusterOptions struct {
	InstanceType   string
	Count          int
	MaxCount       int
	SpotPrice      float32
	PreemptibleVMs bool
	VolumeSize     int

	Acm               bool
	CertManager       bool
	Autoscaler        bool
	KubeDashboard     bool
	KubeDashboardMode string
}

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
	"hybrid": {
		"Hybrid bare-metal Adapter in %s",
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

func CreateKubernetes(kind, name, environment, template string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	nativeRegion, nativeZone, nativeClusterName,
	eksAdmin, azureResourceGroup string,
	options ClusterOptions) {

	err := createK8s(kind, name, environment, template,
		autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun,
		nativeRegion, nativeZone, nativeClusterName,
		eksAdmin, azureResourceGroup,
		options)
	if err != nil {
		log.Fatalf("Unable to create `%s` Kubernetes: %v", kind, err)
	}
}

func createK8s(kind, name, environmentSelector, templateSelector string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	nativeRegion, nativeZone, nativeClusterName,
	eksAdmin, azureResourceGroup string,
	options ClusterOptions) error {

	environment, err := environmentBy(environmentSelector)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Environment: %v", err)
	}
	cloudAccount, err := cloudAccountById(environment.CloudAccount, false)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Cloud Account: %v", err)
	}
	err = verifyClusterCloudAccountKind(kind, cloudAccount)
	if err != nil {
		return err
	}
	name, fqdn, err := verifyClusterBaseDomain(name, cloudAccount)
	if err != nil {
		return err
	}

	platformTag := "platform=" + kind
	if templateSelector == "" && autoCreateTemplate {
		templateSelector = fmt.Sprintf("%s in %s", strings.ToUpper(kind), environment.Name)
	}

	var template *StackTemplate

	if templateSelector != "" && !createNewTemplate {
		template, err = templateBy(templateSelector)
		if err != nil && !strings.HasSuffix(err.Error(), " found") { // TODO proper 404 handling
			return fmt.Errorf("Unable to retrieve cluster Template: %v", err)
		}
		if template != nil && !util.Contains(template.Tags, platformTag) {
			util.Warn("Template `%s` [%s] contain no `%s` tag", template.Name, template.Id, platformTag)
		}
	}

	if template == nil {
		if !autoCreateTemplate {
			return fmt.Errorf("No cluster Template found by `%s`", templateSelector)
		}

		asset := fmt.Sprintf("%s/%s-cluster-template.json.template", requestsBindata, kind)
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
		templateRequest.Tags = []string{platformTag}
		templateRequest.TeamsPermissions = environment.TeamsPermissions // copy permissions from Environment
		templateRequest.ComponentsEnabled = clusterComponents(options)

		template, err = createTemplate(templateRequest)
		if err != nil {
			return fmt.Errorf("Unable to create cluster Template: %v", err)
		}
		err = initTemplate(template.Id)
		if err != nil {
			return fmt.Errorf("Unable to initialize cluster Template: %v", err)
		}
		if config.Verbose {
			log.Printf("Created %s cluster template `%s`", kind, template.Name)
		}
	}

	asset := fmt.Sprintf("%s/%s-cluster-instance.json.template", requestsBindata, kind)
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
		case "dns.name":
			p.Value = name
		case "dns.domain":
			p.Value = fqdn
		case "cloud.region":
			if nativeRegion != "" {
				p.Value = nativeRegion
			} else {
				rm = true
			}
		case "cloud.availabilityZone":
			if nativeZone != "" && !strings.Contains(nativeZone, ",") {
				p.Value = nativeZone
			} else {
				rm = true
			}
		case "cloud.availabilityZones":
			if nativeZone != "" && strings.Contains(nativeZone, ",") {
				p.Value = nativeZone
			} else {
				rm = true
			}
		case "component.kubernetes.eks.admin":
			p.Value = eksAdmin
		case "component.kubernetes.eks.cluster", "component.kubernetes.gke.cluster", "component.kubernetes.aks.cluster":
			p.Value = nativeClusterName
		case "cloud.azureResourceGroupName":
			if azureResourceGroup != "" {
				p.Value = azureResourceGroup
			} else {
				rm = true
			}
		case "component.kubernetes.worker.size", "component.kubernetes.gke.nodeMachineType": // TODO worker.instance.size
			p.Value = options.InstanceType
		case "component.kubernetes.worker.count", "component.kubernetes.gke.minNodeCount":
			p.Value = options.Count
		case "component.kubernetes.worker.maxCount", "component.kubernetes.gke.maxNodeCount":
			if options.MaxCount != 0 {
				p.Value = options.MaxCount
			} else {
				rm = true
			}
		case "component.kubernetes.worker.volume.size":
			if options.VolumeSize != 0 {
				p.Value = options.VolumeSize
			} else {
				rm = true
			}
		case "component.kubernetes.worker.spotPrice": // TODO worker.aws.spotPrice
			if options.SpotPrice > 0 {
				p.Value = options.SpotPrice
			}
		case "component.kubernetes.gke.preemptibleNodes": // TODO worker.gcp.preemptible.enabled
			p.Value = options.PreemptibleVMs
		}
		p = clusterOptions(p, options)
		if !rm {
			parameters = append(parameters, p)
		}
	}
	instanceRequest.Parameters = parameters

	instance, err := createStackInstance(instanceRequest)
	if err != nil {
		return fmt.Errorf("Unable to create cluster Stack Instance: %v", err)
	}

	_, err = commandStackInstance(instance.Id, "deploy", nil, waitAndTailDeployLogs, dryRun)
	if err != nil {
		return fmt.Errorf("Unable to deploy cluster Stack Instance: %v", err)
	}

	return nil
}

func ImportKubernetes(kind, name, environment, template string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	pems io.Reader, clusterBearerToken,
	nativeRegion, nativeZone, nativeEndpoint, nativeClusterName,
	ingressIpOrHost, azureResourceGroup string,
	options ClusterOptions) {

	err := errors.New("Not implemented")
	if importConfig, exist := importConfigs[kind]; exist {
		err = importK8s(importConfig, kind, name, environment, template,
			autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun,
			pems, clusterBearerToken,
			nativeRegion, nativeZone, nativeEndpoint, nativeClusterName,
			ingressIpOrHost, azureResourceGroup,
			options)
	}
	if err != nil {
		log.Fatalf("Unable to import `%s` Kubernetes: %v", kind, err)
	}
}

func importK8s(importConfig ImportConfig, kind, name, environmentSelector, templateSelector string,
	autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun bool,
	pems io.Reader, clusterBearerToken,
	nativeRegion, nativeZone, nativeEndpoint, nativeClusterName,
	ingressIpOrHost, azureResourceGroup string,
	options ClusterOptions) error {

	environment, err := environmentBy(environmentSelector)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Environment: %v", err)
	}
	cloudAccount, err := cloudAccountById(environment.CloudAccount, false)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Cloud Account: %v", err)
	}
	err = verifyClusterCloudAccountKind(kind, cloudAccount)
	if err != nil {
		return err
	}
	name, fqdn, err := verifyClusterBaseDomain(name, cloudAccount)
	if err != nil {
		return err
	}

	ingressIp := ""
	ingressHost := ""

	if kind == "eks" {
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
	} else if kind == "hybrid" {
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
		if err != nil && !strings.HasSuffix(err.Error(), " found") {
			return fmt.Errorf("Unable to retrieve adapter Template: %v", err)
		}
		if template != nil && !util.Contains(template.Tags, adapterTag) {
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
		templateRequest.ComponentsEnabled = clusterComponents(options)

		template, err = createTemplate(templateRequest)
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
		case "cloud.region":
			if nativeRegion != "" {
				p.Value = nativeRegion
			} else {
				rm = true
			}
		case "cloud.availabilityZone":
			if nativeZone != "" {
				p.Value = nativeZone
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
		case "cloud.azureResourceGroupName", "component.kubernetes.aks.resourceGroupName":
			if azureResourceGroup != "" {
				p.Value = azureResourceGroup
			} else {
				rm = true
			}
		}
		p = clusterOptions(p, options)
		if !rm {
			parameters = append(parameters, p)
		}
	}
	instanceRequest.Parameters = parameters

	instance, err := createStackInstance(instanceRequest)
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

	_, err = commandStackInstance(instance.Id, "deploy", nil, waitAndTailDeployLogs, dryRun)
	if err != nil {
		return fmt.Errorf("Unable to deploy adapter Stack Instance: %v", err)
	}

	return nil
}

func verifyClusterCloudAccountKind(clusterKind string, cloudAccount *CloudAccount) error {
	mustBeCloudKind := ""
	switch clusterKind {
	case "k8s-aws", "eks", "metal", "hybrid", "openshift":
		if !strings.HasPrefix(cloudAccount.Kind, "aws") {
			mustBeCloudKind = "AWS"
		}
	case "gke":
		if cloudAccount.Kind != "gcp" {
			mustBeCloudKind = "GCP"
		}
	case "aks":
		if cloudAccount.Kind != "azure" {
			mustBeCloudKind = "Azure"
		}
	}
	if mustBeCloudKind != "" {
		return fmt.Errorf("Cloud Account %s is not %s but `%s`", cloudAccount.BaseDomain, mustBeCloudKind, cloudAccount.Kind)
	}
	return nil
}

func verifyClusterBaseDomain(name string, cloudAccount *CloudAccount) (string, string, error) {
	if strings.Contains(name, ".") {
		suffix := "." + cloudAccount.BaseDomain
		i := strings.LastIndex(name, suffix)
		if !strings.HasSuffix(name, suffix) || i < 1 {
			return "", "", fmt.Errorf("`%s` looks like FQDN, but Cloud Account base domain is `%s`", name, cloudAccount.BaseDomain)
		}
		name = name[:i]
	}
	fqdn := fmt.Sprintf("%s.%s", name, cloudAccount.BaseDomain)
	return name, fqdn, nil
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

func clusterComponents(options ClusterOptions) []string {
	var components []string
	if options.Acm {
		components = append(components, "acm")
	}
	if options.CertManager {
		components = append(components, "cert-manager")
	}
	if options.Autoscaler {
		components = append(components, "cluster-autoscaler")
	}
	if options.KubeDashboard {
		components = append(components, "kube-dashboard")
	}
	return components
}

func clusterOptions(p Parameter, options ClusterOptions) Parameter {
	switch p.Name {
	case "component.kubernetes-dashboard.rbac.kind":
		p.Value = options.KubeDashboardMode
	}
	return p
}
