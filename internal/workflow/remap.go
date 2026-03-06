package workflow

// RemapExecuteWorkflowReferences updates the `workflowId.value` parameter
// in `n8n-nodes-base.executeWorkflow` nodes to point to the correct remote ID.
func RemapExecuteWorkflowReferences(
	nodes []any,
	localIdToName map[string]string,
	remoteNameToId map[string]string,
) []any {
	if len(nodes) == 0 {
		return nodes
	}

	for i, nodeInterface := range nodes {
		node, ok := nodeInterface.(map[string]any)
		if !ok {
			continue
		}

		nodeType, _ := node["type"].(string)
		if nodeType != "n8n-nodes-base.executeWorkflow" {
			continue
		}

		parameters, ok := node["parameters"].(map[string]any)
		if !ok {
			continue
		}

		workflowIdParam, ok := parameters["workflowId"].(map[string]any)
		if !ok {
			// Older versions of n8n might have it directly as a string
			workflowIdStr, isStr := parameters["workflowId"].(string)
			if isStr {
				// Try to map it directly
				if name, exists := localIdToName[workflowIdStr]; exists {
					if remoteId, exists := remoteNameToId[name]; exists {
						parameters["workflowId"] = remoteId
					}
				}
			}
			continue
		}

		// Modern n8n representation
		value, _ := workflowIdParam["value"].(string)
		if value == "" {
			continue
		}

		// Figure out the name of the referenced workflow
		var refName string

		// 1. Best source: cachedResultName (n8n saves the actual name here)
		cachedResultName, _ := workflowIdParam["cachedResultName"].(string)
		if cachedResultName != "" {
			refName = cachedResultName
		} else {
			// 2. Fallback: look up the local ID in our map
			refName = localIdToName[value]
		}

		if refName != "" {
			// Update to the remote ID
			if remoteId, ok := remoteNameToId[refName]; ok {
				workflowIdParam["value"] = remoteId
				// Update cachedResultUrl to match
				workflowIdParam["cachedResultUrl"] = "/workflow/" + remoteId
			}
		}

		// Reassign to parameters and node
		parameters["workflowId"] = workflowIdParam
		node["parameters"] = parameters
		nodes[i] = node
	}

	return nodes
}
