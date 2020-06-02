package multiclient

func stringsMap(strings ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(strings))
	for _, s := range strings {
		result[s] = struct{}{}
	}
	return result
}
