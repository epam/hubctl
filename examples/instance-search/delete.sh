#!/bin/bash -xe
# Copyright (c) 2022 EPAM Systems, Inc.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.


base_domain=smoke-$USER-aws02.epam.devops.delivery
env=Demo02-smoke-$USER

for i in $(seq 1 50); do
    name=eks-$i
    hub api instance sync $name.$base_domain undeployed
    hub api instance delete $name.$base_domain
done
