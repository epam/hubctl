#!/bin/sh
# Copyright (c) 2022 EPAM Systems, Inc.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.


hub api template create < examples/platformless-scratch/template.json
hub api template init S3

hub api instance create < examples/platformless-scratch/instance.json
hub api instance deploy arkadi-s3-01.arkadi-demo01.dev.epam.devops.delivery
