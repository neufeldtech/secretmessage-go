package secretmessage

import (
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func callHealth(url string) error {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (ctl *PublicController) StayAwake() {
	for {
		time.Sleep(5 * time.Minute)
		err := callHealth(ctl.config.AppURL)
		if err != nil {
			ctl.logger.Error("Error calling health endpoint", zap.Error(err))
		}
	}
}
