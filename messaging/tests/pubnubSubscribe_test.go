// Package tests has the unit tests of package messaging.
// pubnubSubscribe_test.go contains the tests related to the Subscribe requests on pubnub Api
package tests

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/pubnub/go/messaging"
	"github.com/pubnub/go/messaging/tests/utils"
	"github.com/stretchr/testify/assert"
)

// TestSubscribeStart prints a message on the screen to mark the beginning of
// subscribe tests.
// PrintTestMessage is defined in the common.go file.
func TestSubscribeStart(t *testing.T) {
	PrintTestMessage("==========Subscribe tests start==========")
}

// TestSubscriptionConnectStatus sends out a subscribe request to a pubnub channel
// and validates the response for the connect status.
func TestSubscriptionConnectStatus(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth(
		"fixtures/subscribe/connectStatus", []string{"uuid"}, 1)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "connectStatus"

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	select {
	case resp := <-successChannel:
		response := fmt.Sprintf("%s", resp)
		if response != "[]" {
			message := "'" + channel + "' connected"
			assert.Contains(response, message)

			close(await)
			return
		}
	case err := <-errorChannel:
		if !IsConnectionRefusedError(err) {
			assert.Fail(string(err))
		}

		close(await)
		return
	case <-timeouts(3):
		assert.Fail("Subscribe timeout 3s")
		close(await)
		return
	}

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestSubscriptionAlreadySubscribed sends out a subscribe request to a pubnub channel
// and when connected sends out another subscribe request. The response for the second
func xTestSubscriptionAlreadySubscribed(t *testing.T) {
	// TODO: this test makes retryCount (module level int) 1 or more that breakes
	// the next test
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/alreadySubscribed", []string{"uuid"}, 1)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "alreadySubscribed"

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	select {
	case resp := <-successChannel:
		response := fmt.Sprintf("%s", resp)
		if response != "[]" {
			message := "'" + channel + "' connected"
			assert.Contains(response, message)
		}
	case err := <-errorChannel:
		if !IsConnectionRefusedError(err) {
			assert.Fail(string(err))
		}
	case <-timeouts(3):
		assert.Fail("Subscribe timeout 3s")
	}

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	select {
	case resp := <-successChannel:
		assert.Fail(fmt.Sprintf(
			"Receive message on success channel, while expecting error: %s",
			string(resp)))
	case err := <-errorChannel:
		assert.Contains(string(err), "already subscribe")
		assert.Contains(string(err), channel)
	case <-timeouts(3):
		assert.Fail("Subscribe timeout 3s")
	}

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestMultiSubscriptionConnectStatus send out a pubnub multi channel subscribe request and
// parses the response for multiple connection status.
func TestMultiSubscriptionConnectStatus(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/connectMultipleStatus", []string{"uuid"}, 1)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	channels := "connectStatus_14,connectStatus_992"
	expectedChannels := strings.Split(channels, ",")
	actualChannels := []string{}
	var actualMu sync.Mutex

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channels, "", successChannel, false, errorChannel)
	go func() {

		for {
			select {
			case resp := <-successChannel:
				var response []interface{}

				err := json.Unmarshal(resp, &response)
				if err != nil {
					assert.Fail(err.Error())
				}

				assert.Len(response, 3)
				assert.Contains(response[1].(string), "Subscription to channel")
				assert.Contains(response[1].(string), "connected")

				actualMu.Lock()
				actualChannels = append(actualChannels, response[2].(string))
				l := len(actualChannels)
				actualMu.Unlock()

				if l == 2 {
					await <- true
					return
				}

			case err := <-errorChannel:
				if !IsConnectionRefusedError(err) {
					assert.Fail(string(err))
				}

				await <- false
			case <-timeouts(5):
				assert.Fail("Subscribe timeout 3s")
				await <- false
			}

		}
	}()

	select {
	case <-await:
		actualMu.Lock()
		assert.True(utils.AssertStringSliceElementsEqual(expectedChannels, actualChannels),
			fmt.Sprintf("%s(expected) should be equal to %s(actual)", expectedChannels, actualChannels))
		actualMu.Unlock()
	case <-timeouts(10):
		assert.Fail("Timeout 5s")
	}

	go pubnubInstance.Unsubscribe(channels, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channels, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestSubscriptionForSimpleMessage first subscribes to a pubnub channel and then publishes
// a message on the same pubnub channel. The subscribe response should receive this same message.
func TestSubscriptionForSimpleMessage(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/forSimpleMessage", []string{"uuid"}, 2)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "subscriptionConnectedForSimple"
	customMessage := "Test message"

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	go func() {
		for {
			select {
			case resp := <-successChannel:
				response := fmt.Sprintf("%s", resp)
				if response != "[]" {
					message := "'" + channel + "' connected"

					if strings.Contains(response, message) {

						successChannel := make(chan []byte)
						errorChannel := make(chan []byte)

						go pubnubInstance.Publish(channel, customMessage,
							successChannel, errorChannel)
						select {
						case <-successChannel:
						case err := <-errorChannel:
							assert.Fail(string(err))
						case <-timeout():
							assert.Fail("Publish timeout")
						}
					} else {
						assert.Contains(response, customMessage)

						close(await)
						return
					}
				}
			case err := <-errorChannel:
				if !IsConnectionRefusedError(err) {
					assert.Fail(string(err))
				}

				close(await)
				return
			case <-timeouts(3):
				assert.Fail("Subscribe timeout 3s")
				close(await)
				return
			}

		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestSubscriptionForSimpleMessageWithCipher first subscribes to a pubnub channel and then publishes
// an encrypted message on the same pubnub channel. The subscribe response should receive
// the decrypted message.
func TestSubscriptionForSimpleMessageWithCipher(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/forSimpleMessageWithCipher", []string{"uuid"}, 2)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "enigma", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "subscriptionConnectedForSimpleWithCipher"
	customMessage := "Test message"

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	go func() {
		for {
			select {
			case resp := <-successChannel:
				response := fmt.Sprintf("%s", resp)
				if response != "[]" {
					message := "'" + channel + "' connected"

					if strings.Contains(response, message) {

						successChannel := make(chan []byte)
						errorChannel := make(chan []byte)

						go pubnubInstance.Publish(channel, customMessage,
							successChannel, errorChannel)
						select {
						case <-successChannel:
						case err := <-errorChannel:
							assert.Fail(string(err))
						case <-timeout():
							assert.Fail("Publish timeout")
						}
					} else {
						assert.Contains(response, customMessage)

						close(await)
						return
					}
				}
			case err := <-errorChannel:
				if !IsConnectionRefusedError(err) {
					assert.Fail(string(err))
				}

				close(await)
				return
			case <-timeouts(3):
				assert.Fail("Subscribe timeout 3s")
				close(await)
				return
			}
		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestSubscriptionForComplexMessage first subscribes to a pubnub channel and then publishes
// a complex message on the same pubnub channel. The subscribe response should receive
// the same message.
func TestSubscriptionForComplexMessage(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/forComplexMessage", []string{"uuid"}, 2)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "subscriptionConnectedForComplex"
	customComplexMessage := InitComplexMessage()

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	go func() {
		for {
			select {
			case resp := <-successChannel:
				response := fmt.Sprintf("%s", resp)
				if response != "[]" {
					message := "'" + channel + "' connected"

					if strings.Contains(response, message) {

						successChannel := make(chan []byte)
						errorChannel := make(chan []byte)

						go pubnubInstance.Publish(channel, customComplexMessage,
							successChannel, errorChannel)
						select {
						case <-successChannel:
						case err := <-errorChannel:
							assert.Fail(string(err))
						case <-timeout():
							assert.Fail("Publish timeout")
						}
					} else {
						var arr []interface{}
						err := json.Unmarshal(resp, &arr)
						if err != nil {
							assert.Fail(err.Error())
						} else {
							assert.True(CheckComplexData(arr))
						}

						close(await)
						return
					}
				}
			case err := <-errorChannel:
				if !IsConnectionRefusedError(err) {
					assert.Fail(string(err))
				}

				close(await)
				return
			case <-timeouts(3):
				assert.Fail("Subscribe timeout 3s")
				close(await)
				return
			}

		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// TestSubscriptionForComplexMessageWithCipher first subscribes to a pubnub channel and then publishes
// an encrypted complex message on the same pubnub channel. The subscribe response should receive
// the decrypted message.
func TestSubscriptionForComplexMessageWithCipher(t *testing.T) {
	assert := assert.New(t)

	stop := NewVCRBoth("fixtures/subscribe/forComplexMessageWithCipher", []string{"uuid"}, 2)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "enigma", false, "")

	messaging.SetSubscribeTimeout(10)

	channel := "subscriptionConnectedForComplexWithCipher"
	customMessage := InitComplexMessage()

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, "", successChannel, false, errorChannel)
	go func() {
		for {
			select {
			case resp := <-successChannel:
				response := fmt.Sprintf("%s", resp)
				if response != "[]" {
					message := "'" + channel + "' connected"

					if strings.Contains(response, message) {

						successChannel := make(chan []byte)
						errorChannel := make(chan []byte)

						go pubnubInstance.Publish(channel, customMessage,
							successChannel, errorChannel)
						select {
						case <-successChannel:
						case err := <-errorChannel:
							assert.Fail(string(err))
						case <-timeout():
							assert.Fail("Publish timeout")
						}
					} else {
						var arr []interface{}
						err := json.Unmarshal(resp, &arr)
						if err != nil {
							assert.Fail(err.Error())
						} else {
							assert.True(CheckComplexData(arr))
						}

						close(await)
						return
					}
				}
			case err := <-errorChannel:
				if !IsConnectionRefusedError(err) {
					assert.Fail(string(err))
				}

				close(await)
				return
			case <-timeouts(3):
				assert.Fail("Subscribe timeout 3s")
				close(await)
				return
			}
		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
}

// ValidateComplexData takes an interafce as a parameter and iterates through it
// It validates each field of the response interface against the initialized struct
// CustomComplexMessage, ReplaceEncodedChars and InitComplexMessage are defined in the common.go file.
func ValidateComplexData(m map[string]interface{}) bool {
	//m := vv.(map[string]interface{})
	//if val,ok := m["VersionId"]; ok {
	//fmt.Println("VersionId",m["VersionId"])
	//}
	customComplexMessage := InitComplexMessage()
	valid := false
	for k, v := range m {
		// fmt.Println("k:", k, "v:", v)
		if k == "OperationName" {
			if m["OperationName"].(string) == customComplexMessage.OperationName {
				valid = true
			} else {
				//fmt.Println("OperationName")
				return false
			}
		} else if k == "VersionID" {
			if a, ok := v.(string); ok {
				verID, convErr := strconv.ParseFloat(a, 64)
				if convErr != nil {
					//fmt.Println(convErr)
					return false
				}
				if float32(verID) == customComplexMessage.VersionID {
					valid = true
				} else {
					//fmt.Println("VersionID")
					return false
				}
			}
		} else if k == "TimeToken" {
			i, convErr := strconv.ParseInt(v.(string), 10, 64)
			if convErr != nil {
				//fmt.Println(convErr)
				return false
			}
			if i == customComplexMessage.TimeToken {
				valid = true
			} else {
				//fmt.Println("TimeToken")
				return false
			}
		} else if k == "DemoMessage" {
			b1 := v.(map[string]interface{})
			jsonData, _ := json.Marshal(customComplexMessage.DemoMessage.DefaultMessage)
			if val, ok := b1["DefaultMessage"]; ok {
				if val.(string) != string(jsonData) {
					//fmt.Println("DefaultMessage")
					return false
				}
				valid = true
			}
		} else if k == "SampleXML" {
			data := &Data{}
			//s1, _ := url.QueryUnescape(m["SampleXML"].(string))
			s1, _ := m["SampleXML"].(string)

			reader := strings.NewReader(ReplaceEncodedChars(s1))
			err := xml.NewDecoder(reader).Decode(&data)

			if err != nil {
				//fmt.Println(err)
				return false
			}
			jsonData, _ := json.Marshal(customComplexMessage.SampleXML)
			if s1 == string(jsonData) {
				valid = true
			} else {
				//fmt.Println("SampleXML")
				return false
			}
		} else if k == "Channels" {
			strSlice1, _ := json.Marshal(v)
			strSlice2, _ := json.Marshal(customComplexMessage.Channels)
			//s1, err := url.QueryUnescape(string(strSlice1))
			s1 := string(strSlice1)
			/*if err != nil {
				fmt.Println(err)
				return false
			}*/
			if s1 == string(strSlice2) {
				valid = true
			} else {
				//fmt.Println("Channels")
				return false
			}
		}
	}
	return valid
}

// CheckComplexData iterates through the json interafce and will read when
// map type is encountered.
// CustomComplexMessage and InitComplexMessage are defined in the common.go file.
func CheckComplexData(b interface{}) bool {
	valid := false
	switch vv := b.(type) {
	case string:
		//fmt.Println( "is string", vv)
	case int:
		//fmt.Println( "is int", vv)
	case []interface{}:
		//fmt.Println( "is an array:")
		//for i, u := range vv {
		for _, u := range vv {
			return CheckComplexData(u)
			//fmt.Println(i, u)
		}
	case map[string]interface{}:
		m := vv
		return ValidateComplexData(m)
	default:
	}
	return valid
}

// TestMultipleResponse publishes 2 messages and then parses the response
// by creating a subsribe request with a timetoken prior to publishing of the messages
// on subscribing we will get one response with multiple messages which should be split into
// 2 by the client api.
func TestMultipleResponse(t *testing.T) {
	SendMultipleResponse(t, false, "multipleResponse", "")
}

// TestMultipleResponseEncrypted publishes 2 messages and then parses the response
// by creating a subsribe request with a timetoken prior to publishing of the messages
// on subscribing we will get one response with multiple messages which should be split into
// 2 by the clinet api.
func TestMultipleResponseEncrypted(t *testing.T) {
	SendMultipleResponse(t, true, "multipleResponseEncrypted", "enigma")
}

// SendMultipleResponse is the common implementation for TestMultipleResponsed and
// TestMultipleResponseEncrypted
func SendMultipleResponse(t *testing.T, encrypted bool, testName, cipher string) {
	assert := assert.New(t)

	stop := NewVCRBoth(fmt.Sprintf("fixtures/subscribe/%s", testName), []string{"uuid"}, 1)
	defer stop()

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", cipher, false, "")
	channel := "Channel_MultipleResponse"

	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	retTime := GetServerTimeString("time_uuid")
	fmt.Println("time is", retTime)

	message1 := "message1"
	message2 := "message2"

	successChannelPublish := make(chan []byte)
	errorChannelPublish := make(chan []byte)

	go pubnubInstance.Publish(channel, message1, successChannelPublish,
		errorChannelPublish)
	select {
	case <-successChannelPublish:
	case err := <-errorChannelPublish:
		assert.Fail("Publish #1 error", string(err))
	case <-timeout():
		assert.Fail("Publish #1 timeout")
	}

	successChannelPublish2 := make(chan []byte)
	errorChannelPublish2 := make(chan []byte)

	go pubnubInstance.Publish(channel, message2, successChannelPublish2,
		errorChannelPublish2)
	select {
	case <-successChannelPublish2:
	case err := <-errorChannelPublish2:
		assert.Fail("Publish #2 error", string(err))
	case <-timeout():
		assert.Fail("Publish #2 timeout")
	}

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnubInstance.Subscribe(channel, retTime, successChannel, false, errorChannel)

	go func() {
		messageCount := 0

		for {
			select {
			case value := <-successChannel:
				response := fmt.Sprintf("%s", value)
				message := "'" + channel + "' connected"
				if strings.Contains(response, message) {
					continue
				} else {
					var s []interface{}
					err := json.Unmarshal(value, &s)
					if err == nil {
						if len(s) > 0 {
							if message, ok := s[0].([]interface{}); ok {
								if messageT, ok2 := message[0].(string); ok2 {
									if (len(message) > 0) && (messageT == message1) {
										messageCount++
									}
									if (len(message) > 0) && (messageT == message2) {
										messageCount++
									}
								}
							}
						}
					}

					if messageCount >= 2 {
						await <- true
						return
					}
				}

			case err := <-errorChannel:
				assert.Fail("Subscribe error", string(err))
				await <- false
				return
			case <-timeout():
				assert.Fail("Subscribe timeout")
				await <- false
				return
			}
		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(channel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, channel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)

	pubnubInstance.CloseExistingConnection()
}

// TestResumeOnReconnectFalse upon reconnect, it should use a 0 (zero) timetoken.
// This has the effect of continuing from “this moment onward”.
// Any messages received since the previous timeout or network error are skipped
func xTestResumeOnReconnectFalse(t *testing.T) {
	messaging.SetResumeOnReconnect(false)

	r := GenRandom()
	assert := assert.New(t)
	pubnubChannel := fmt.Sprintf("testChannel_subror_%d", r.Intn(20))
	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	messaging.SetSubscribeTimeout(3)

	go pubnubInstance.Subscribe(pubnubChannel, "", successChannel, false, errorChannel)
	for {
		select {
		case <-successChannel:
		case value := <-errorChannel:
			if string(value) != "[]" {
				newPubnubTest := &messaging.PubnubUnitTest{}

				assert.Equal("0", newPubnubTest.GetSentTimeToken(pubnubInstance))
			}
			return
		case <-messaging.Timeouts(60):
			assert.Fail("Subscribe timeout")
			return
		}
	}

	messaging.SetSubscribeTimeout(310)

	go pubnubInstance.Unsubscribe(pubnubChannel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, pubnubChannel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)

	pubnubInstance.CloseExistingConnection()
}

// TestResumeOnReconnectTrue upon reconnect, it should use the last successfully retrieved timetoken.
// This has the effect of continuing, or “catching up” to missed traffic.
func TestResumeOnReconnectTrue(t *testing.T) {
	messaging.SetResumeOnReconnect(true)

	r := GenRandom()
	assert := assert.New(t)
	pubnubChannel := fmt.Sprintf("testChannel_subror_%d", r.Intn(20))
	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	messaging.SetSubscribeTimeout(3)

	go pubnubInstance.Subscribe(pubnubChannel, "", successChannel, false, errorChannel)
	for {
		select {
		case <-successChannel:
		case value := <-errorChannel:
			if string(value) != "[]" {
				newPubnubTest := &messaging.PubnubUnitTest{}

				assert.Equal(newPubnubTest.GetTimeToken(pubnubInstance), newPubnubTest.GetSentTimeToken(pubnubInstance))
			}
			return
		case <-messaging.Timeouts(60):
			assert.Fail("Subscribe timeout")
			return
		}
	}

	messaging.SetSubscribeTimeout(310)

	go pubnubInstance.Unsubscribe(pubnubChannel, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, pubnubChannel, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)

	pubnubInstance.CloseExistingConnection()
}

// TestMultiplexing tests the multiplexed subscribe request.
func TestMultiplexing(t *testing.T) {
	SendMultiplexingRequest(t, "TestMultiplexing", false, false)
}

// TestMultiplexing tests the multiplexed subscribe request wil ssl.
func TestMultiplexingSSL(t *testing.T) {
	SendMultiplexingRequest(t, "TestMultiplexingSSL", true, false)
}

// TestMultiplexing tests the encrypted multiplexed subscribe request.
func TestEncryptedMultiplexing(t *testing.T) {
	SendMultiplexingRequest(t, "TestEncryptedMultiplexing", false, true)
}

// TestMultiplexing tests the encrypted multiplexed subscribe request with ssl.
func TestEncryptedMultiplexingWithSSL(t *testing.T) {
	SendMultiplexingRequest(t, "TestEncryptedMultiplexingWithSSL", true, true)
}

// SendMultiplexingRequest is the common method to test TestMultiplexing,
// TestMultiplexingSSL, TestEncryptedMultiplexing, TestEncryptedMultiplexingWithSSL.
//
// It subscribes to 2 channels in the same request and then calls the ParseSubscribeMultiplexedResponse
// for further processing.
//
// Parameters:
// t: *testing.T instance.
// testName: testname for display.
// ssl: ssl setting.
// encrypted: encryption setting.
func SendMultiplexingRequest(t *testing.T, testName string, ssl bool, encrypted bool) {
	assert := assert.New(t)

	cipher := ""
	if encrypted {
		cipher = "enigma"
	}
	message1 := "message1"
	message2 := "message2"

	pubnubChannel1, pubnubChannel2 := GenerateTwoRandomChannelStrings(1)

	pubnubChannels := pubnubChannel1 + "," + pubnubChannel2

	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", cipher, ssl, "")

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)

	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	await := make(chan bool)

	go pubnubInstance.Subscribe(pubnubChannels, "", successChannel, false, errorChannel)

	go func() {
		messageCount := 0
		channelCount := 0

		for {
			select {
			case value := <-successChannel:

				if string(value) != "[]" {
					response := fmt.Sprintf("%s", value)

					message := "' connected"

					messageT1 := "'" + pubnubChannel1 + "' connected"
					messageT2 := "'" + pubnubChannel2 + "' connected"

					if strings.Contains(response, message) {
						if strings.Contains(response, messageT1) {
							channelCount++
						}

						if strings.Contains(response, messageT2) {
							channelCount++
						}

						if channelCount >= 2 {
							returnPublishChannel := make(chan []byte)
							errorChannelPub := make(chan []byte)

							go pubnubInstance.Publish(pubnubChannel1, message1, returnPublishChannel, errorChannelPub)
							select {
							case <-returnPublishChannel:
							case err := <-errorChannelPub:
								assert.Fail(string(err))
								return
							case <-timeout():
								assert.Fail("Publish msg#1 timeout")
							}

							returnPublishChannel2 := make(chan []byte)
							errorChannelPub2 := make(chan []byte)

							go pubnubInstance.Publish(pubnubChannel2, message2, returnPublishChannel2, errorChannelPub2)
							select {
							case <-returnPublishChannel2:
							case err := <-errorChannelPub2:
								assert.Fail(string(err))
								return
							case <-timeout():
								assert.Fail("Publish msg#2 timeout")
							}
						}
					} else {
						var s []interface{}
						err := json.Unmarshal(value, &s)
						if err == nil {
							if len(s) > 2 {
								if message, ok := s[0].([]interface{}); ok {
									if messageT, ok2 := message[0].(string); ok2 {

										if (len(message) > 0) && (messageT == message1) && (s[2].(string) == pubnubChannel1) {
											messageCount++
										}
										if (len(message) > 0) && (messageT == message2) && (s[2].(string) == pubnubChannel2) {
											messageCount++
										}
									}
								}
							}
						}

						if messageCount >= 2 {
							await <- true
							return
						}
					}
				}
			case err := <-errorChannel:
				assert.Fail(string(err))
			case <-messaging.SubscribeTimeout():
				assert.Fail("Subscribe timeout")
			}
		}
	}()

	<-await

	go pubnubInstance.Unsubscribe(pubnubChannels, unsubscribeSuccessChannel, unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, pubnubChannels, "", unsubscribeSuccessChannel, unsubscribeErrorChannel)
	pubnubInstance.CloseExistingConnection()
}

// TestSubscribeEnd prints a message on the screen to mark the end of
// subscribe tests.
// PrintTestMessage is defined in the common.go file.
func TestSubscribeEnd(t *testing.T) {
	PrintTestMessage("==========Subscribe tests end==========")
}
