package service

import (
	"encoding/json"
	"net/url"
)

type ServiceDefinition struct {
	Name     string   `json:"name"`
	Display  string   `json:"display"`
	Audience string   `json:"audience"`
	Redirect *url.URL `json:"redirect"`
}

func (s *ServiceDefinition) UnmarshalJSON(
	data []byte,
) error {
	type Alias ServiceDefinition
	tmp := &struct {
		Redirect string `json:"redirect"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	redirect, err := url.Parse(tmp.Redirect)
	if err != nil {
		return err
	}
	s.Redirect = redirect
	return nil
}
