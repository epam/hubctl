#!/bin/sh

hub api template create < examples/platformless-scratch/template.json
hub api template init S3

hub api instance create < examples/platformless-scratch/instance.json
hub api instance deploy arkadi-s3-01.arkadi-demo01.dev.superhub.io
