#!/bin/bash

(cd cmd/pixelserve && GOOS=linux GOARCH=386 go build)
(cd cmd/pixelsend && GOOS=linux GOARCH=386 go build)
scp cmd/pixelserve/pixelserve cmd/pixelsend/pixelsend ./*.tpl ./*.png kiki.prod:pixel
scp prod.yaml kiki.prod:pixel/pixelcheck.yaml

