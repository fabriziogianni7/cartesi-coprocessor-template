package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"dapp/common"
)

var (
	infolog = log.New(os.Stderr, "[ info ]  ", log.Lshortfile)
	errlog  = log.New(os.Stderr, "[ error ] ", log.Lshortfile)
)

func HandleAdvance(data *rollups.AdvanceResponse) error {
	// Log the received advance request data
	dataMarshal, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("HandleAdvance: failed marshaling json: %w", err)
	}
	infolog.Println("Received advance request data", string(dataMarshal))

	// Decode the payload from hex to string
	payloadStr, err := rollups.Hex2Str(data.Payload)
	if err != nil {
		return fmt.Errorf("HandleAdvance: failed to decode hex payload: %w", err)
	}

	// Parse the payload into a map
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return fmt.Errorf("HandleAdvance: failed to unmarshal payload: %w", err)
	}
	infolog.Println("Parsed payload:", payload)

	// Check if the method is "increment" and the counter exists
	if method, ok := payload["method"].(string); ok && method == "increment" {
		if counter, ok := payload["counter"].(float64); ok {
			newCounter := int(counter) + 1
			infolog.Printf("Counter incremented to: %d", newCounter)

			// Convert the new counter to a hex string
			counterHex := fmt.Sprintf("0x%064x", newCounter)

			// Send a notice with the updated counter
			notice := &rollups.NoticeRequest{
				Payload: counterHex,
			}

			_, err := rollups.SendNotice(notice)
			if err != nil {
				return fmt.Errorf("HandleAdvance: failed to send notice: %w", err)
			}

			return nil
		}
	}

	infolog.Println("Invalid method or missing counter value in payload")
	return nil
}

func Handler(response *rollups.FinishResponse) error {
	var err error

	switch response.Type {
	default:
		data := new(rollups.AdvanceResponse)
		if err = json.Unmarshal(response.Data, data); err != nil {
			return fmt.Errorf("Handler: Error unmarshaling advance: %s", err)
		}
		err = HandleAdvance(data)
	}
	return err
}

func main() {
	finish := rollups.FinishRequest{Status: "accept"}

	for {
		infolog.Println("Sending finish")
		res, err := rollups.SendFinish(&finish)
		if err != nil {
			errlog.Panicln("Error: error making http request: ", err)
		}
		infolog.Println("Received finish status ", strconv.Itoa(res.StatusCode))

		if res.StatusCode == 202 {
			infolog.Println("No pending rollup request, trying again")
		} else {

			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				errlog.Panicln("Error: could not read response body: ", err)
			}

			var response rollups.FinishResponse
			err = json.Unmarshal(resBody, &response)
			if err != nil {
				errlog.Panicln("Error: unmarshaling body:", err)
			}

			finish.Status = "accept"
			err = Handler(&response)
			if err != nil {
				errlog.Println(err)
				finish.Status = "reject"
			}
		}
	}
}
