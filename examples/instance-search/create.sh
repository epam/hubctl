#!/bin/bash -xe
# Copyright (c) 2022 EPAM Systems, Inc.
# 
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.


base_domain=smoke-$USER-aws02.superhub.io
env=Demo02-smoke-$USER

traefik_versions=(1.7.21 2.1.9)
kube_dashboard_versions=(2.0.0-rc3 2.0.0-rc6)

for i in $(seq 1 50); do
    name=eks-$i
    hub api cluster create eks $name t3.medium 1 3 -e $env -m EKS --kube-dashboard --eks-admin arkadi -s 0.05 -y
    t=${traefik_versions[$(($RANDOM % 2))]}
    d=${kube_dashboard_versions[$(($RANDOM % 2))]}
    cat $(dirname $0)/patch.json |
        sed -e s/\\\$traefik_version/$t/ |
        sed -e s/\\\$kube_dashboard_version/$d/ |
            hub api instance patch $name.$base_domain
done
