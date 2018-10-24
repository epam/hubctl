#!/bin/sh

hub api template create < examples/platform/template.json
hub api template init Pg

hub api instance create < examples/platform/instance.json
hub api instance deploy pg@arkadi309.arkadi308.kubernetes.delivery
