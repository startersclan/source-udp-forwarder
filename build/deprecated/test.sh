#!/bin/sh
workspaceFolder=$( git rev-parse --show-toplevel )
echo "${workspaceFolder}"
