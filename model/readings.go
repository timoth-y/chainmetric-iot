package model

import "github.com/timoth-y/iot-blockchain-contracts/models"

type SensorReading struct {
	Source string
	Value float64
}

type SensorsReadingResults map[models.Metric] float64

type SensorReadingsPipe map[models.Metric] chan SensorReading
