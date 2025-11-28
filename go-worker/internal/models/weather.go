package models

import (
	"encoding/json"
	"errors"
	"strconv"
)

// WeatherMsg represents a weather message from the queue
type WeatherMsg struct {
	Timestamp          string                 `json:"timestamp"`
	Temperatura        float64                `json:"temperatura"`
	Umidade            float64                `json:"umidade"`
	Vento              float64                `json:"vento"`
	Ceu                float64                `json:"ceu"`
	ProbabilidadeChuva float64                `json:"probabilidade_chuva"`
	raw                map[string]interface{} `json:"-"`
}

// NormalizeAndValidate creates a WeatherMsg from raw JSON data
func NormalizeAndValidate(raw map[string]interface{}) (*WeatherMsg, error) {
	msg := &WeatherMsg{raw: raw}

	getString := func(k string) (string, bool) {
		v, ok := raw[k]
		if !ok || v == nil {
			return "", false
		}
		switch t := v.(type) {
		case string:
			return t, true
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64), true
		case json.Number:
			return t.String(), true
		default:
			b, _ := json.Marshal(v)
			return string(b), true
		}
	}

	getFloat := func(k string) (float64, bool) {
		v, ok := raw[k]
		if !ok || v == nil {
			return 0, false
		}
		switch t := v.(type) {
		case float64:
			return t, true
		case string:
			f, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return 0, false
			}
			return f, true
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				return 0, false
			}
			return f, true
		default:
			return 0, false
		}
	}

	if ts, ok := getString("timestamp"); ok {
		msg.Timestamp = ts
	} else {
		return nil, errors.New("missing 'timestamp'")
	}

	if f, ok := getFloat("temperatura"); ok {
		msg.Temperatura = f
	} else {
		return nil, errors.New("missing or invalid 'temperatura'")
	}

	if f, ok := getFloat("umidade"); ok {
		msg.Umidade = f
	} else {
		return nil, errors.New("missing or invalid 'umidade'")
	}

	if f, ok := getFloat("vento"); ok {
		msg.Vento = f
	} else {
		return nil, errors.New("missing or invalid 'vento'")
	}

	if f, ok := getFloat("probabilidade_chuva"); ok {
		msg.ProbabilidadeChuva = f
	} else if f, ok := getFloat("precipitation_probability"); ok {
		msg.ProbabilidadeChuva = f
	} else {
		return nil, errors.New("missing or invalid 'probabilidade_chuva'")
	}

	return msg, nil
}