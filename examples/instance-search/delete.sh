#!/bin/bash -xe

base_domain=smoke-$USER-aws02.superhub.io
env=Demo02-smoke-$USER

for i in $(seq 1 50); do
    name=eks-$i
    hub api instance sync $name.$base_domain undeployed
    hub api instance delete $name.$base_domain
done
