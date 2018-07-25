package pubnub

import (
	"fmt"
	"net/url"
	"testing"

	h "github.com/pubnub/go/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestGrantRequestBasic(t *testing.T) {
	assert := assert.New(t)

	opts := &grantOpts{
		AuthKeys:      []string{"my-auth-key"},
		Channels:      []string{"ch"},
		ChannelGroups: []string{"cg"},
		Read:          true,
		Write:         true,
		Manage:        true,
		TTL:           5000,
		setTTL:        true,
		pubnub:        pubnub,
	}

	path, err := opts.buildPath()
	assert.Nil(err)
	u := &url.URL{
		Path: path,
	}

	h.AssertPathsEqual(t,
		fmt.Sprintf("/v1/auth/grant/sub-key/%s", opts.pubnub.Config.SubscribeKey),
		u.EscapedPath(), []int{})

	query, err := opts.buildQuery()
	assert.Nil(err)

	expected := &url.Values{}
	expected.Set("auth", "my-auth-key")
	expected.Set("channel", "ch")
	expected.Set("channel-group", "cg")
	expected.Set("r", "1")
	expected.Set("w", "1")
	expected.Set("m", "1")
	expected.Set("ttl", "5000")
	h.AssertQueriesEqual(t, expected, query,
		[]string{"pnsdk", "uuid", "timestamp"}, []string{})

	body, err := opts.buildBody()

	assert.Nil(err)
	assert.Equal([]byte{}, body)
}

func TestGrantOptsValidateSub(t *testing.T) {
	assert := assert.New(t)
	pn := NewPubNub(NewDemoConfig())
	pn.Config.SubscribeKey = ""
	opts := &grantOpts{
		AuthKeys:      []string{"my-auth-key"},
		Channels:      []string{"ch"},
		ChannelGroups: []string{"cg"},
		Read:          true,
		Write:         true,
		Manage:        true,
		TTL:           5000,
		setTTL:        true,
		pubnub:        pn,
	}

	assert.Equal("pubnub/validation: pubnub: \x15: Missing Subscribe Key", opts.validate().Error())
}

func TestGrantOptsValidateSec(t *testing.T) {
	assert := assert.New(t)
	pn := NewPubNub(NewDemoConfig())
	pn.Config.SecretKey = ""
	opts := &grantOpts{
		AuthKeys:      []string{"my-auth-key"},
		Channels:      []string{"ch"},
		ChannelGroups: []string{"cg"},
		Read:          true,
		Write:         true,
		Manage:        true,
		TTL:           5000,
		setTTL:        true,
		pubnub:        pn,
	}

	assert.Equal("pubnub/validation: pubnub: \x15: Missing Secret Key", opts.validate().Error())
}

func TestGrantOptsValidatePub(t *testing.T) {
	assert := assert.New(t)
	pn := NewPubNub(NewDemoConfig())
	pn.Config.PublishKey = ""
	opts := &grantOpts{
		AuthKeys:      []string{"my-auth-key"},
		Channels:      []string{"ch"},
		ChannelGroups: []string{"cg"},
		Read:          true,
		Write:         true,
		Manage:        true,
		TTL:           5000,
		setTTL:        true,
		pubnub:        pn,
	}

	assert.Equal("pubnub/validation: pubnub: \x15: Missing Publish Key", opts.validate().Error())
}

func TestNewGrantResponseErrorUnmarshalling(t *testing.T) {
	assert := assert.New(t)
	jsonBytes := []byte(`s`)

	_, _, err := newGrantResponse(jsonBytes, StatusResponse{})
	assert.Equal("pubnub/parsing: Error unmarshalling response: {s}", err.Error())
}

func TestNewGrantResponseManageEnabled(t *testing.T) {
	assert := assert.New(t)
	jsonBytes := []byte(`{"message":"Success","payload":{"level":"channel-group+auth","subscribe_key":"sub-c-b9ab9508-43cf-11e8-9967-869954283fb4","ttl":1440,"r":1,"m":1,"w":1,"channels":{"ch1":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}},"ch2":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}},"ch3":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}}},"channel-groups":{"cg1":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}},"cg2":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0,"ttl":1},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}},"cg3":{"auths":{"my-auth-key-1":{"r":1,"w":1,"m":1,"d":0},"my-auth-key-2":{"r":1,"w":1,"m":1,"d":0}}}}},"service":"Access Manager","status":200}`)

	_, _, err := newGrantResponse(jsonBytes, StatusResponse{})

	assert.Nil(err)
}

func TestNewGrantResponseManageEnabledInv(t *testing.T) {
	assert := assert.New(t)
	jsonBytes := []byte(`{"message":"Success","payload":{"level":"channel-group+auth","subscribe_key":"sub-c-b9ab9508-43cf-11e8-9967-869954283fb4","ttl":0,"r":0,"m":0,"w":0,"channels":{"ch1":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}},"ch2":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}},"ch3":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}}},"channel-groups":{"cg1":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1,"ttl":4},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}},"cg2":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1,"ttl":6},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}},"cg3":{"auths":{"my-auth-key-1":{"r":0,"w":0,"m":0,"d":1},"my-auth-key-2":{"r":0,"w":0,"m":0,"d":1}}}}},"service":"Access Manager","status":200}`)

	_, _, err := newGrantResponse(jsonBytes, StatusResponse{})

	assert.Nil(err)
}

func TestNewGrantResponseManageEnabledCH(t *testing.T) {
	assert := assert.New(t)
	jsonBytes := []byte(`{"message":"Success","payload":{"level":"user","subscribe_key":"sub-c-b9ab9508-43cf-11e8-9967-869954283fb4","ttl":1440,"channel":"ch1","auths":{"my-pam-key":{"r":1,"w":1,"m":0,"d":0}}},"service":"Access Manager","status":200}`)

	_, _, err := newGrantResponse(jsonBytes, StatusResponse{})

	assert.Nil(err)
}

func TestNewGrantResponseManageEnabledCHM(t *testing.T) {
	assert := assert.New(t)
	jsonBytes := []byte(`{"message":"Success","payload":{"level":"user","subscribe_key":"sub-c-b9ab9508-43cf-11e8-9967-869954283fb4","ttl":1440,"channel":"ch1","auths":{"my-pam-key":{"r":1,"w":1,"m":1,"d":0}}},"service":"Access Manager","status":200}`)

	_, _, err := newGrantResponse(jsonBytes, StatusResponse{})

	assert.Nil(err)
}

func TestGrantTTL(t *testing.T) {
	assert := assert.New(t)
	pn := NewPubNub(NewDemoConfig())
	gb := newGrantBuilder(pn)
	gb.TTL(10)
	assert.Equal(10, gb.opts.TTL)
}
