package common

func CheckMap(m map[string]string, keys ...string) (vals []string, allOk bool, noKey string) {
	vals = make([]string, 0, len(m))
	for _, noKey = range keys {
		val, ok := m[noKey]
		if !ok {
			return
		}
		vals = append(vals, val)
	}
	allOk = true
	return
}
