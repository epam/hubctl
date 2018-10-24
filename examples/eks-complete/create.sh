#!/bin/sh

hub api template create < examples/eks-complete/template.json
hub api template init 1

hub api instance create < examples/eks-complete/instance.json
hub api instance deploy 1
