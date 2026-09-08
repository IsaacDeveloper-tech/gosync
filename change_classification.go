package main

func classifyAdditionsAndDeletions(comparisons []EntryComparison) []SynchronizationAction {
	actions := make([]SynchronizationAction, 0)
	for _, comparison := range comparisons {
		switch {
		case comparison.First != nil && comparison.Second == nil:
			if comparison.Confirmed == nil {
				actions = append(actions, SynchronizationAction{
					Kind:         SynchronizationActionCopyToSecond,
					RelativePath: comparison.RelativePath,
				})
				continue
			}

			actions = append(actions, SynchronizationAction{
				Kind:         SynchronizationActionDeleteFromFirst,
				RelativePath: comparison.RelativePath,
			})
		case comparison.First == nil && comparison.Second != nil:
			if comparison.Confirmed == nil {
				actions = append(actions, SynchronizationAction{
					Kind:         SynchronizationActionCopyToFirst,
					RelativePath: comparison.RelativePath,
				})
				continue
			}

			actions = append(actions, SynchronizationAction{
				Kind:         SynchronizationActionDeleteFromSecond,
				RelativePath: comparison.RelativePath,
			})
		}
	}

	return actions
}
