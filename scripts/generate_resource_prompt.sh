#!/bin/bash

resourceName=""
resourceType=""
resourceRender=""

if [ $# -eq 3 ]; then
	resourceName=$1
	resourceType=$2
	resourceRender=`cat $3`
else

	echo -n "What is the name of the resource to generate: "
	read resourceName
	echo -n "What is kubernetes type: "
	read resourceType
	echo "Example Kubectl output of that type (press Ctrl+D when done):"

	while IFS= read -r line; do
  	if [[ "$line" == $'\x04' ]]; then # Check for literal Ctrl+D (EOT)
    	break  # Exit the loop
  	fi
  	resourceRender+="$line"
  	if [[ -n "$line" ]]; then #Add newline only if line is not empty
      	resourceRender+=$'\n'
  	fi
	done
fi

echo "----------------------------- BEGIN LLM PROMPT"
echo "I will give you an example file for kubernetes Services that describe a ServiceExtra, ServiceRenderer, and ServiceWatcher."
echo "You will generate a similar file but instead of a kubernetes Service you will support a kubernetes $resouceName of type $resourceType."
echo "The render function will geneerate output similar to how Kubectl might, and an example of what that output looks like is this: "
echo "Please note that the ServiceExtra struct has a Copy method that makes a deep copy of the struct in such away that changes to any of it's values won't affect the original value.  We do this because when the state of the Service changes we make a copy and then change some values, including values in maps or slices and we don't want those changes to propogate to previous iterations of the struct."

cat prompt.txt
echo "--- Begin Example Output ---"
echo "$resourceRender"
echo "--- End Example Output ---"
echo "--- Begin Example service.go file ---"
cat ../internal/resources/service.go
echo "--- End Example service.go file ---"
