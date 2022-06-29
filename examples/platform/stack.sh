#!/bin/sh
# Copyright (c) 2022 EPAM Systems, Inc.
# 
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.


hub api template create < examples/platform/template.json
hub api template init Pg

hub api instance create < examples/platform/instance.json
hub api instance deploy pg@arkadi309.arkadi308.kubernetes.delivery
