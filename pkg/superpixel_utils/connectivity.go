package superpixel_utils

import (
	"github.com/segtile/segtile/pkg/superpixel_map"
)

// EnforceConnectivity ensures that every label forms a single 4-connected component.
// If a label appears in multiple disjoint components, the largest component keeps the
// label, and every smaller component is merged into the label most common along its border.
// Labels are renumbered to remain contiguous afterwards.
func EnforceConnectivity(labelMap *superpixel_map.LabelMap) (changed bool) {

	// Step 1: Connected components in label map
	componentIdMap, componentLabel, componentSize := FindConnectedComponents(labelMap)
	numComponents := len(componentLabel)

	// If each label has exactly one connected component, no merges are needed.
	winnerOfLabel := make(map[uint32]int, numComponents)
	for id := 0; id < numComponents; id++ {
		label := componentLabel[id]
		if currentWinner, ok := winnerOfLabel[label]; !ok || componentSize[id] > componentSize[currentWinner] {
			winnerOfLabel[label] = id
		}
	}
	if len(winnerOfLabel) == numComponents {
		return false
	}

	// Step 2: Relationships between connected components
	componentBorders := LabelMapNeighborhoodsWeighted(componentIdMap)

	// Store the relationships in compact adjacency arrays, indexed by unshifted component id.
	adjacencyDegree := make([]int, numComponents)
	for id := 0; id < numComponents; id++ {
		adjacencyDegree[id] = len(componentBorders[uint32(id+1)])
	}
	adjacencyOffset := make([]int, numComponents+1)
	for id := 0; id < numComponents; id++ {
		adjacencyOffset[id+1] = adjacencyOffset[id] + adjacencyDegree[id]
	}
	adjacencyNeighbor := make([]int, adjacencyOffset[numComponents])
	adjacencyWeight := make([]int, adjacencyOffset[numComponents])
	for id := 0; id < numComponents; id++ {
		position := adjacencyOffset[id]
		for neighborID, sharedBorderLength := range componentBorders[uint32(id+1)] {
			adjacencyNeighbor[position] = int(neighborID) - 1
			adjacencyWeight[position] = sharedBorderLength
			position++
		}
	}

	// Step 3: the largest component of each label survives with that label; every other
	// component sharing the label is a loser that must be merged elsewhere.
	finalLabel := make([]uint32, numComponents)
	finalized := make([]bool, numComponents)
	pending := make([]int, 0, numComponents)
	for id := 0; id < numComponents; id++ {
		if winnerOfLabel[componentLabel[id]] == id {
			finalLabel[id] = componentLabel[id]
			finalized[id] = true
		} else {
			pending = append(pending, id)
		}
	}

	// Step 4: repeatedly merge every pending component that borders an already-finalized
	// one into whichever finalized neighbor shares the longest border (ties broken by the
	// lowest label, for determinism). Once a component is finalized it is never revisited.
	for len(pending) > 0 {
		stillPending := pending[:0]
		progressed := false
		for _, id := range pending {
			bestWeight := -1
			bestTarget := -1
			for position := adjacencyOffset[id]; position < adjacencyOffset[id+1]; position++ {
				neighborId := adjacencyNeighbor[position]
				if !finalized[neighborId] {
					continue
				}
				weight := adjacencyWeight[position]
				if weight > bestWeight || (weight == bestWeight && bestTarget != -1 && finalLabel[neighborId] < finalLabel[bestTarget]) {
					bestWeight = weight
					bestTarget = neighborId
				}
			}
			if bestTarget == -1 {
				stillPending = append(stillPending, id)
				continue
			}
			finalLabel[id] = finalLabel[bestTarget]
			finalized[id] = true
			progressed = true
		}

		// A group of minority components that only border each other has no finalized neighbor to attach to.
		// Break the deadlock deterministically by letting the largest of them survive with its own label, then keep resolving the rest.
		if !progressed {
			largest := stillPending[0]
			for _, id := range stillPending[1:] {
				if componentSize[id] > componentSize[largest] {
					largest = id
				}
			}
			finalLabel[largest] = componentLabel[largest]
			finalized[largest] = true
			remaining := stillPending[:0]
			for _, id := range stillPending {
				if id != largest {
					remaining = append(remaining, id)
				}
			}
			stillPending = remaining
		}

		pending = stillPending
	}

	// Step 5: put the resolved labels back to the label map.
	changed = false

	for y := 0; y < labelMap.Height; y++ {
		for x := 0; x < labelMap.Width; x++ {
			id := componentIdMap.ValueAt(x, y) - 1
			newLabel := finalLabel[id]
			if newLabel == componentLabel[id] {
				continue
			}
			changed = true
			labelMap.SetValueAt(x, y, newLabel)
		}
	}

	return changed
}
