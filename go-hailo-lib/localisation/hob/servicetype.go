package hob

import (
	"time"
)

type ServiceTypeStatus string

const (
	ServiceTypeStatusEnabled  = ServiceTypeStatus("ENABLED")
	ServiceTypeStatusDisabled = ServiceTypeStatus("DISABLED")
	ServiceTypeStatusBeta     = ServiceTypeStatus("BETA")
)

type GoingHome string

const (
	GoingHomeDisabled = GoingHome("DISABLED")
	GoingHomeAlways   = GoingHome("ALWAYS")
	GoingHomePriority = GoingHome("PRIORITY")
)

type I18nText struct {
	Id            string `json:"id"`
	Language      string `json:"language"`
	Text          string `json:"text"`
	IsDefaultLang bool   `json:"isDefaultLang"`
}

type JobSettings struct {
	Enabled                    bool      `json:"enabled"`
	CashJobs                   bool      `json:"cashJobs"`
	CardPayment                bool      `json:"cardPayment"`
	CardToCashSwitch           bool      `json:"cardToCashSwitch"`
	FareScreenType             uint32    `json:"fareScreenType"` // fareTypeMeterTip=0, fareTypeMeterTipTollsNoTipOnTolls=1, fareTypeFixedPriceTipTollsNoTipOnTolls=2
	FixedToVirtualMeterEnabled bool      `json:"fixedToVirtualEnabled"`
	ShowDriverAllocatedAlert   bool      `json:"showDriverAllocatedAlert"`
	SoonToClear                bool      `json:"soonToClear"`
	PriorityEnabled            bool      `json:"priorityEnabled"` // @DEPRECATED remove in favour of struct
	Priority                   Priority  `json:"priority"`
	GoingHome                  GoingHome `json:"goingHome" enum:"DISABLED,ALWAYS,PRIORITY" description:"How do we determine if going home is available"`
}

type Priority struct {
	Enabled          bool   `json:"enabled"`
	MorningPeakText  string `json:"morningPeakText"`
	EveningPeakText  string `json:"eveningPeakText"`
	PeakTimeReminder bool   `json:"peak_time_reminder"`
}

type Icons struct {
	SelectorActive   string `json:"selectorActive"`
	SelectorInactive string `json:"selectorInactive"`
	PrebookList      string `json:"prebookList"`
}

type JsonDuration string

func (jsonDuration JsonDuration) Duration() time.Duration {
	duration, err := time.ParseDuration(string(jsonDuration))
	if err == nil {
		return duration
	}
	return time.Duration(0)
}

type ServiceType struct {
	Id                           string                  `json:"id" readOnly:"true"`
	Name                         string                  `json:"name"`
	Status                       ServiceTypeStatus       `json:"status" enum:"BETA,ENABLED,DISABLED"`
	Tier                         uint32                  `json:"tier" description:"Preference order on PMUH screen"`
	FreeWaitingTime              JsonDuration            `json:"freeWaitingTime"`
	MaxUnverifiedFare            float64                 `json:"maxUnverifiedFare"` // i.e. 50.00
	MinAcceptableFare            float64                 `json:"minAcceptableFare"` // i.e. 2.20
	MinUnverifiedFare            float64                 `json:"minUnverifiedFare"` //i.e. 2.50
	MaxFare                      float64                 `json:"maxFare"`           // i.e. 999.00
	MinFare                      float64                 `json:"minFare"`           // i.e. 2.50
	FastPayFeePercentage         uint32                  `json:"fastpayFeePercentage"`
	FastPayMode                  uint32                  `json:"fastpayMode" enum:"0,1,2" description:"FAST_PAY_DISABLED = 0, FAST_PAY_DRIVER_PAYS = 1, FAST_PAY_CUSTOMER_PAYS = 2" ` // FAST_PAY_DISABLED = 0, FAST_PAY_DRIVER_PAYS = 1, FAST_PAY_CUSTOMER_PAYS = 2
	HailoJobSettings             JobSettings             `json:"hailoJobSettings"`
	TollsAndExtras               bool                    `json:"tollsAndExtras"`
	AutoOffShift                 JsonDuration            `json:"autoOffShift"`
	VirtualMeter                 bool                    `json:"virtualMeter" description:"whether the service type uses a virtual meter"`
	FixedVirtual                 bool                    `json:"fixedVirtual" description:"Use fixed fare pricing to calculate virtual meter rate from POB to job end"`
	PercentOverTaxiRate          int32                   `json:"percentOverTaxiRate"`
	MinEta                       int32                   `json:"minEta"`
	MaxEta                       int32                   `json:"maxEta"`
	JobHistoryEnabled            bool                    `json:"jobHistoryEnabled"`
	JobHistoryDeleteEnabled      bool                    `json:"jobHistoryDeleteEnabled"`
	AlwaysShowDestinationOnOffer bool                    `json:"alwaysShowDestinationOnOffer"`
	AssumableServiceTypes        []*AssumableServiceType `json:"assumableServiceTypes"`
	GoingHomeEnabled             bool                    `json:"goingHomeEnabled"`
	Icons                        Icons                   `json:"icons"`
	VehicleOptions               VehicleOptions          `json:"vehicleOptions"`
}

type ServiceTypes map[string]*ServiceType

func (serviceTypes ServiceTypes) Ids() []string {
	result := make([]string, 0)
	for serviceTypeId, _ := range serviceTypes {
		result = append(result, serviceTypeId)
	}
	return result
}

type AssumableServiceType struct {
	Id                  string `json:"id"`                  // service type
	AlwaysAssume        bool   `json:"alwaysAssume"`        // Always assume this service type
	AssumeForNewDrivers bool   `json:"assumeForNewDrivers"` // Assume for new drivers
}
