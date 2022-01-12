package controllers

import (
	"fmt"
	"strings"
)

// Fargate nodes queried from the API server have an extra ".000~"
// appended to the end, but the API server does not recognize
// that extra substring when performing Get or Describe ops.
func TrimNodeName(s string) (k string) {
	// Fargate node names are formatted like:
	//    <node-name>.ec2.internal.<resource-id>
	// We want to strip away '.<resource-id>

	subs := strings.SplitAfter(s, ".")

	for _, v := range subs {
		if strings.Contains(v, ".") {
			k = fmt.Sprintf("%s%s", k, v)
		}
	}

	// Remove the trailing .
	k = strings.TrimRight(k, ".")

	return
}
