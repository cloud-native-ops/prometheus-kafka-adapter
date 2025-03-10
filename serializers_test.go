package main

import (
	"math"
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
)

func NewWriteRequest() *prompb.WriteRequest {
	return &prompb.WriteRequest{
		Timeseries: []*prompb.TimeSeries{
			{
				Labels: []*prompb.Label{
					{Name: "__name__", Value: "foo"},
					{Name: "labelfoo", Value: "label-bar"},
				},
				Samples: []prompb.Sample{
					{Timestamp: 0, Value: 456},
					{Timestamp: 10000, Value: math.Inf(1)},
				},
			},
		},
	}
}

func TestSerializeEmptyTimeseriesToJSON(t *testing.T) {
	request := &prompb.WriteRequest{}
	serializer, err := NewJSONSerializer()
	assert.Nil(t, err)

	data, err := Serialize(serializer, request)
	assert.Nil(t, err)
	assert.Len(t, data, 0)
	assert.NotNil(t, data)
}

func TestSerializeToJSON(t *testing.T) {
	serializer, err := NewJSONSerializer()
	assert.Nil(t, err)

	writeRequest := NewWriteRequest()
	output, err := Serialize(serializer, writeRequest)
	assert.Len(t, output["metrics"], 2)
	assert.Nil(t, err)

	expectedSamples := []string{
		"{\"value\":\"456\",\"timestamp\":\"1970-01-01T00:00:00Z\",\"name\":\"foo\",\"labels\":{\"__name__\":\"foo\",\"labelfoo\":\"label-bar\"}}",
		"{\"value\":\"+Inf\",\"timestamp\":\"1970-01-01T00:00:10Z\",\"name\":\"foo\",\"labels\":{\"__name__\":\"foo\",\"labelfoo\":\"label-bar\"}}",
	}

	for i, metric := range output["metrics"] {
		assert.JSONEqf(t, expectedSamples[i], string(metric[:]), "wrong json serialization found")
	}
}

func TestSerializeEmptyTimeseriesToAvroJSON(t *testing.T) {
	request := &prompb.WriteRequest{}
	serializer, err := NewAvroJSONSerializer("schemas/metric.avsc")
	assert.Nil(t, err)

	data, err := Serialize(serializer, request)
	assert.Nil(t, err)
	assert.Len(t, data, 0)
	assert.NotNil(t, data)
}

func TestSerializeToAvro(t *testing.T) {
	serializer, err := NewAvroJSONSerializer("schemas/metric.avsc")
	assert.Nil(t, err)

	writeRequest := NewWriteRequest()
	output, err := Serialize(serializer, writeRequest)
	assert.Len(t, output["metrics"], 2)
	assert.Nil(t, err)

	expectedSamples := []string{
		"{\"value\":\"456\",\"timestamp\":\"1970-01-01T00:00:00Z\",\"name\":\"foo\",\"labels\":{\"__name__\":\"foo\",\"labelfoo\":\"label-bar\"}}",
		"{\"value\":\"+Inf\",\"timestamp\":\"1970-01-01T00:00:10Z\",\"name\":\"foo\",\"labels\":{\"__name__\":\"foo\",\"labelfoo\":\"label-bar\"}}",
	}

	for i, metric := range output["metrics"] {
		assert.JSONEqf(t, expectedSamples[i], string(metric[:]), "wrong json serialization found")
	}
}

func TestTemplatedTopic(t *testing.T) {
	var err error
	topicTemplate, err = parseTopicTemplate("{{ index . \"labelfoo\" | replace \"bar\" \"foo\" | substring 6 -1 }}")
	assert.Nil(t, err)
	serializer, err := NewJSONSerializer()
	assert.Nil(t, err)

	writeRequest := NewWriteRequest()
	output, err := Serialize(serializer, writeRequest)

	for k := range output {
		assert.Equal(t, "foo", k, "templated topic failed")
	}
}

func TestFilter(t *testing.T) {
	rulesText := `['foo{y="2"}','foo', 'bar{x="1"}',
'up{x="1",y="2"}', 'baz{key="valu
e1;value2"}','bar{y="2"}']`

	rules, _ := parseMatchList(rulesText)
	for _, mf := range rules {
		match[mf.GetName()] = mf
	}
	type TestCase struct {
		Name   string
		Labels map[string]string
		Expect bool
	}

	testList := []TestCase{
		{Name: "foo", Labels: map[string]string{"z": "3"}, Expect: true},
		{Name: "bar", Labels: map[string]string{"x": "1"}, Expect: true},
		{Name: "bar", Labels: map[string]string{"x": "2"}, Expect: false},
		{Name: "bar", Labels: map[string]string{"y": "2"}, Expect: true},
		{Name: "bar", Labels: map[string]string{"y": "1"}, Expect: false},
		{Name: "up", Labels: map[string]string{"x": "1", "y": "2"}, Expect: true},
		{Name: "up", Labels: map[string]string{"x": "1", "y": "2", "z": "3"}, Expect: true},
		{Name: "up", Labels: map[string]string{"x": "2", "y": "1"}, Expect: false},
		{Name: "go", Labels: map[string]string{"x": "1", "y": "2"}, Expect: false},
	}

	for _, tcase := range testList {
		assert.Equal(t, tcase.Expect, filter(tcase.Name, tcase.Labels))
	}
}

func BenchmarkSerializeToAvroJSON(b *testing.B) {
	serializer, _ := NewAvroJSONSerializer("schemas/metric.avsc")
	writeRequest := NewWriteRequest()

	for n := 0; n < 20000; n++ {
		Serialize(serializer, writeRequest)
	}
}

func BenchmarkSerializeToJSON(b *testing.B) {
	serializer, _ := NewJSONSerializer()
	writeRequest := NewWriteRequest()

	for n := 0; n < 20000; n++ {
		Serialize(serializer, writeRequest)
	}
}
