package sensors

import (
	"fmt"
	"time"

	"github.com/timoth-y/iot-blockchain-contracts/models"

	"github.com/timoth-y/iot-blockchain-sensorsys/drivers/peripherals"
	"github.com/timoth-y/iot-blockchain-sensorsys/drivers/sensor"
	"github.com/timoth-y/iot-blockchain-sensorsys/model/metrics"
)

var (
	InterruptMode      byte = 0
	InterruptThreshold byte = 0
	SamplingRate       byte = CCS811_DRIVE_MODE_1SEC
)

type CCS811 struct {
	*peripherals.I2C
}

func NewCCS811(addr uint16, bus int) sensor.Sensor {
	return &CCS811{
		I2C: peripherals.NewI2C(addr, bus),
	}
}

func (s *CCS811) ID() string {
	return "CCS811"
}

func (s *CCS811) Init() (err error) {
	if err = s.I2C.Init(); err != nil {
		return
	}

	if !s.Verify() {
		return fmt.Errorf("not that sensorType")
	}

	err = s.setReset()
	time.Sleep(CCS811_RESET_TIME * time.Millisecond)

	_, err = s.getStatus()

	err = s.WriteBytes(CCS811_BOOTLOADER_APP_START); if err != nil {
		return err
	}

	time.Sleep(CCS811_APP_START_TIME * time.Millisecond)

	status, err := s.getStatus(); if err != nil {
		return err
	}

	if status & CCS811_ERROR_BIT != 0 {
		return fmt.Errorf("CCS811 device has error")
	}

	if status & CCS811_FW_MODE_BIT == 0 {
		return fmt.Errorf("CCS811 device is in FW mode")
	}

	err = s.setConfig()

	return
}

func (s *CCS811) Read() (eCO2 float64, eTVOC float64, err error) {
	retry := 10
	for retry > 0 {
		retry--
		ready, err := s.isDataReady(); if err != nil {
			return 0, 0, err
		}
		if ready {
			buffer, err := s.ReadRegBytes(CCS811_ALG_RESULT_DATA, 4)
			if err != nil {
				return 0, 0, err
			}
			eCO2 = float64((uint16(buffer[0]) << 8) | uint16(buffer[1]))
			eTVOC = float64((uint16(buffer[2]) << 8) | uint16(buffer[3]))
			break
		}
		time.Sleep(CCS811_RETRY_TIME * time.Millisecond)
	}
	err = nil
	return
}

func (s *CCS811) Harvest(ctx *sensor.Context) {
	eCO2, eTVOC, err := s.Read()

	if eCO2 != 0 {
		ctx.For(metrics.AirCO2Concentration).Write(eCO2)
	}

	if eTVOC != 0 {
		ctx.For(metrics.AirTVOCsConcentration).Write(eTVOC)
	}

	ctx.Error(err)
}

func (s *CCS811) Metrics() []models.Metric {
	return []models.Metric {
		metrics.AirCO2Concentration,
		metrics.AirTVOCsConcentration,
	}
}

func (s *CCS811) Verify() bool {
	buffer, err := s.ReadReg(CCS811_HW_ID)
	if err == nil && buffer == CCS811_HW_ID_CODE {
		return true
	}

	return false
}

func (s *CCS811) isDataReady() (bool, error) {
	sts, err := s.getStatus()
	if err != nil {
		return false, err
	}

	return (sts & CCS811_DATA_READY_BIT) != 0, nil
}

func (s *CCS811) getStatus() (byte, error) {
	data, err := s.ReadReg(CCS811_STATUS); if err != nil {
		return 0, err
	}

	return data, nil
}

func (s *CCS811) setConfig() error {
	buffer := make([]byte, 1)
	bin1 := 0x01 & InterruptThreshold
	bin2 := 0x01 & InterruptMode
	bin3 := 0x03 & SamplingRate
	buffer[0] = bin1 << 2 | bin2 << 3 | bin3 << 4

	return s.WriteRegBytes(CCS811_MEAS_MODE, buffer...)
}

func (s *CCS811) setReset() error {
	return s.WriteRegBytes(CCS811_SW_RESET, 0x11, 0xE5, 0x72, 0x8A)
}
