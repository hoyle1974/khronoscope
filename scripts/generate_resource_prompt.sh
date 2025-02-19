#!/bin/bash

echo -n "What is the name of the resource to generate: "
read resourceName
echo -n "What is kubernetes type: "
read resourceType
echo "Example Kubectl output of that type (press Ctrl+D when done):"

# Read multi-line input into resourceRender
resourceRender=""
while IFS= read -r line; do
  resourceRender+="$line"$'\n'
done

echo "----------------------------- BEGIN LLM PROMPT"
echo "I will give you an example file for kubernetes Services that describe a ServiceExtra, ServiceRenderer, and ServiceWatcher."
echo "You will generate a similar file but instead of a kubernetes Service you will support a kubernetes $resouceName of type $resourceType."
echo "The render function will geneerate output similar to how Kubectl might, and an example of what that output looks like is this: "
echo "Please note that the ServiceExtra struct has a Copy method that makes a deep copy of the struct in such away that changes to any of it's values won't affect the original value.  We do this because when the state of the Service changes we make a copy and then change some values, including values in maps or slices and we don't want those changes to propogate to previous iterations of the struct."
echo "In "github.com/hoyle1974/khronoscope/internal/misc" package I have some helper functions you will use:"
echo "func FormatArray(arr []string) string // converts an array into a comma-separated string"
echo "func DeepCopyArray[K any](s []K) []K // Performs a deep copy of an array"
echo "func DeepCopyMap[K comparable, V any](m map[K]V) map[K]V // Performs a deep copy of a map"
echo "func RenderMapOfStrings[V any](name string, t map[string]V) []string // Converts a map to an array of strings with the contents of the map printed in sorted order"
echo "func Range[K comparable, V any](m map[K]V) func(func(K, V) bool) // Iterates over a map in sorted key order"
echo "--- Begin Example Output ---"
echo $resourceRenderer
echo "--- End Example Output ---"
echo "--- Begin Example service.go file ---"
cat ../internal/resources/service.go
echo "--- End Example service.go file ---"
