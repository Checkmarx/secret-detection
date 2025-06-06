package report

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
