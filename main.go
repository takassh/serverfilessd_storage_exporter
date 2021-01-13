package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ResponseJSON is struct of Jupyterhub response for /hub/api/users
type ResponseJSON []struct {
	Name         string `json:"name"`
	Server       string `json:"server"`
	LastActivity string `json:"last_activity"`
}

var (
	apiHost  = flag.String("host", "http://localhost:8888/hub/api", "API host")
	willStop = flag.Bool("stop", true, "stop single server")
	apiToken = flag.String("token", "", "jupyterhub token (admin)")
	waitHour = flag.Int("hours", 24, "hours to wait for stop server")
)

const (
	namespace   = "storage"
	metricsPath = "/metrics"
)

type myCollector struct{}

var (
	directorySizeDesc = prometheus.NewDesc(
		"directory_size",
		"Current used directory size(MB).",
		[]string{"directoryName"}, nil,
	)
)

func (cc myCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

//GetDirectorySize ...
func GetDirectorySize() map[string]int {
	cmd := "du -d 2 /mnt/ssd/gpu/workspace 2>/dev/null | sort -hr"
	out, _ := exec.Command("sh", "-c", cmd).Output()
	output := string(out)
	lines := strings.Split(output, "\n")
	directorySizes := map[string]int{}
	for i := 0; i < len(lines)-1; i++ {
		line := strings.Split(lines[i], "\t")
		tmp := strings.Split(line[1], "/")
		k, _ := strconv.Atoi(line[0])
		directorySizes[tmp[len(tmp)-1]] = k / (1024 * 2)
	}

	return directorySizes
}

func (cc myCollector) Collect(ch chan<- prometheus.Metric) {
	directorySizes := GetDirectorySize()

	for directoryName, size := range directorySizes {
		ch <- prometheus.MustNewConstMetric(
			directorySizeDesc,
			prometheus.UntypedValue,
			float64(size),
			directoryName,
		)
	}
}

func main() {
	flag.Parse()

	reg := prometheus.NewPedanticRegistry()
	cc := myCollector{}
	prometheus.WrapRegistererWithPrefix(namespace+"_", reg).MustRegister(cc)

	http.Handle(metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Directory Size Exporter</title></head>
			<body>
			<h1>Directory Size Exporter</h1>
			<p><a href="` + metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(":9225", nil))
}
