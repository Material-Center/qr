#!/bin/bash

npm run build
adb push dist/* /storage/emulated/0/脚本/
