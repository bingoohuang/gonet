package gonet

import "net/url"

// BuildURL 创建一个url
func BuildURL(base string, queryParams map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()

	for k, v := range queryParams {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}
