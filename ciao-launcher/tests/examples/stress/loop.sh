#!/bin/bash

for i in $(seq -f "%g" 0 9)
do
  ciaolc startf start_container-$i.yaml
done


