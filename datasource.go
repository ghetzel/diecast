package diecast

type DataSource struct {
	ID         string      `yaml:"id"`
	URL        string      `yaml:"url"`
	Transforms interface{} `yaml:"transform,omitempty"`
}

type DataSet []DataSource

func (self DataSet) Refresh() (map[string]interface{}, error) {
	return nil, ErrNotImplemented
}
