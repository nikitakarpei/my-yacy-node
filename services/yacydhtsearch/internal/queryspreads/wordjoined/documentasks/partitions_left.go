package documentasks

import "slices"

type partitionsLeft []uint

func (partitions partitionsLeft) without(settledPartitions []uint) partitionsLeft {
	return slices.DeleteFunc(partitions, func(partition uint) bool {
		return slices.Contains(settledPartitions, partition)
	})
}
