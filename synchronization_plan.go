package main

type SynchronizationPlan struct {
	Actions []SynchronizationAction
}

func generateSynchronizationPlan(
	comparisons []EntryComparison,
	directoryContents map[string][]string,
	requestFileConflictSide func(EntryComparison) (SynchronizationSide, error),
	requestFileDirectoryConflictSide func(EntryComparison, []string) (SynchronizationSide, error),
) (SynchronizationPlan, error) {
	actions := make([]SynchronizationAction, 0)
	for _, comparison := range comparisons {
		if comparison.First == nil || comparison.Second == nil {
			actions = append(actions, classifyAdditionsAndDeletions([]EntryComparison{comparison})...)
			continue
		}

		if comparison.First.Kind == EntryKindFile && comparison.Second.Kind == EntryKindFile {
			var requestConflictSide func() (SynchronizationSide, error)
			if requestFileConflictSide != nil {
				requestConflictSide = func() (SynchronizationSide, error) {
					return requestFileConflictSide(comparison)
				}
			}
			action, required, err := resolveFileContentConflict(comparison, requestConflictSide)
			if err != nil {
				return SynchronizationPlan{}, err
			}
			if required {
				actions = append(actions, action)
			}
			continue
		}

		if comparison.First.Kind != comparison.Second.Kind {
			var requestConflictSide func([]string) (SynchronizationSide, error)
			if requestFileDirectoryConflictSide != nil {
				requestConflictSide = func(contents []string) (SynchronizationSide, error) {
					return requestFileDirectoryConflictSide(comparison, contents)
				}
			}
			action, required, err := resolveFileDirectoryConflict(
				comparison,
				directoryContents[comparison.RelativePath],
				requestConflictSide,
			)
			if err != nil {
				return SynchronizationPlan{}, err
			}
			if required {
				actions = append(actions, action)
			}
		}
	}

	return SynchronizationPlan{Actions: actions}, nil
}
