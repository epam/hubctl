package metrics

type DDSeries struct {
	Metric string
	Type   string `json:",omitempty"`
	Host   string `json:",omitempty"`
	// Interval int
	Tags   []string `json:",omitempty"`
	Points [][]int64
}

/*
	"metric": "hubcli.commands.usage",
	"type": "count",
	"host": "714cbf9b-f8df-4362-8aea-b7321ba33a2e",
	"tags": ["command:hub-elaborate", "status:success", "machine-id:714cbf9b-f8df-4362-8aea-b7321ba33a2e"],
	"points": [
	  	[
			$NOW,
			1
		]
	]
*/

type DDSeriesResponse struct {
	Status string
}
