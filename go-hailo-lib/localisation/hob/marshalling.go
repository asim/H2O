package hob

import (
	"time"

	hobpb "github.com/HailoOSS/hob-service/proto"
)

func ProtoToJobSettings(j *hobpb.JobSettings) JobSettings {
	if j == nil {
		return JobSettings{}
	}

	return JobSettings{
		Enabled:                    j.GetEnabled(),
		CashJobs:                   j.GetCashJobs(),
		CardPayment:                j.GetCardPayments(),
		CardToCashSwitch:           j.GetCardToCashSwitch(),
		FareScreenType:             j.GetFareScreenType(),
		FixedToVirtualMeterEnabled: j.GetFixedToVirtualEnabled(),
		ShowDriverAllocatedAlert:   j.GetShowDriverAllocatedAlert(),
		SoonToClear:                j.GetSoonToClear(),
		PriorityEnabled:            j.GetPriorityEnabled(),
		Priority:                   ProtoToPriority(j.GetPriority()),
		GoingHome:                  GoingHome(j.GetGoingHome()),
	}
}

func ProtoToServiceType(s *hobpb.ServiceType) *ServiceType {
	if s == nil {
		return &ServiceType{}
	}

	return &ServiceType{
		Id:                           s.GetId(),
		Name:                         s.GetName(),
		Status:                       ServiceTypeStatus(s.GetStatus().String()),
		Tier:                         s.GetTier(),
		FreeWaitingTime:              JsonDuration((time.Duration(s.GetFreeWaitingSeconds()) * time.Second).String()),
		MaxUnverifiedFare:            s.GetMaxUnverifiedFare(),
		MinAcceptableFare:            s.GetMinAcceptableFare(),
		MinUnverifiedFare:            s.GetMinUnverifiedFare(),
		MaxFare:                      s.GetMaxFare(),
		MinFare:                      s.GetMinFare(),
		FastPayFeePercentage:         s.GetFastPayFeePercentage(),
		FastPayMode:                  s.GetFastPayMode(),
		HailoJobSettings:             ProtoToJobSettings(s.GetHailoJobSettings()),
		TollsAndExtras:               s.GetTollsAndExtras(),
		AutoOffShift:                 JsonDuration((time.Duration(s.GetAutoOffshiftSec()) * time.Second).String()),
		VirtualMeter:                 s.GetVirtualMeter(),
		FixedVirtual:                 s.GetFixedVirtual(),
		PercentOverTaxiRate:          s.GetPercentOverTaxiRate(),
		MinEta:                       s.GetMinEta(),
		MaxEta:                       s.GetMaxEta(),
		JobHistoryEnabled:            s.GetJobHistoryEnabled(),
		JobHistoryDeleteEnabled:      s.GetJobHistoryDeleteEnabled(),
		AlwaysShowDestinationOnOffer: s.GetAlwaysShowDestinationOnOffer(),
		AssumableServiceTypes:        ProtoToAssumableServiceTypes(s.GetAssumableServiceTypes()),
		GoingHomeEnabled:             s.GetGoingHomeEnabled(),
		Icons:                        ProtoToIcons(s.GetIcons()),
		VehicleOptions:               ProtoToVehicleOptions(s.GetVehicleOptions()),
	}
}

func ProtoToVehicleOptions(vehicleOptions *hobpb.OverrideVehicleOptions) VehicleOptions {
	if vehicleOptions == nil {
		return VehicleOptions{}
	}

	return VehicleOptions{
		Accessible: vehicleOptions.GetAccessible(),
		Passengers: vehicleOptions.GetPassengers(),
	}
}

func ProtoToIcons(icons *hobpb.Icons) Icons {
	if icons == nil {
		return Icons{}
	}

	return Icons{
		SelectorActive:   icons.GetSelectorActive(),
		SelectorInactive: icons.GetSelectorInactive(),
		PrebookList:      icons.GetPrebookList(),
	}
}

func ProtoToPriority(p *hobpb.Priority) Priority {
	if p == nil {
		return Priority{}
	}

	return Priority{
		Enabled:          p.GetEnabled(),
		MorningPeakText:  p.GetMorningPeakText(),
		EveningPeakText:  p.GetEveningPeakText(),
		PeakTimeReminder: p.GetPeakTimeReminder(),
	}
}

func ProtoToAssumableServiceTypes(types []*hobpb.AssumableServiceType) []*AssumableServiceType {
	if types == nil {
		return nil
	}

	res := make([]*AssumableServiceType, len(types))

	for i, ast := range types {
		res[i] = &AssumableServiceType{
			Id:                  ast.GetId(),
			AlwaysAssume:        ast.GetAlwaysAssume(),
			AssumeForNewDrivers: ast.GetAssumeForNewDrivers(),
		}
	}

	return res
}

func ProtoToHob(s *hobpb.Hob) *Hob {
	if s == nil {
		return &Hob{}
	}

	return &Hob{
		Code:   s.GetCode(),
		Name:   s.GetName(),
		Status: HobStatus(s.GetStatus().String()),
		Country: Country{
			Cctld:      s.GetCountry().GetCctld(),
			ISO_3166_1: s.GetCountry().GetIso_3166_1(),
		},
		Currency: s.GetCurrency(),
		Language: s.GetLanguage(),
		GeoInfo: GeoInfo{
			Centroid: Centroid{
				Lat: s.GetGeoInfo().GetCentroid().GetLat(),
				Lng: s.GetGeoInfo().GetCentroid().GetLng(),
			},
			Minimum: Location{
				Lat: s.GetGeoInfo().GetMinimum().GetLat(),
				Lng: s.GetGeoInfo().GetMinimum().GetLng(),
			},
			Maximum: Location{
				Lat: s.GetGeoInfo().GetMaximum().GetLat(),
				Lng: s.GetGeoInfo().GetMaximum().GetLng(),
			},
		},
		Phone: Phone{
			CallingCode: s.GetPhone().GetCallingCode(),
			TrunkPrefix: s.GetPhone().GetTrunkPrefix(),
		},
		Timezone:      s.GetTimezone(),
		DefaultLocale: s.GetDefaultLocale(),
		Locale:        s.GetLocale(),
		Misc: Misc{
			DriverTermsUrl:      s.GetMisc().GetDriverTermsUrl(),
			FallbackOnOsm:       s.GetMisc().GetFallbackOnOsm(),
			EnableProdDebugMenu: s.GetMisc().GetEnableProdDebugMenu(),
			DistanceDisplayUnit: s.GetMisc().GetDistanceDisplayUnit(),
			PayWithHailo:        s.GetMisc().GetPayWithHailo(),
			ShowStats:           s.GetMisc().GetShowStats(),
			ShowRouteToPickup:   s.GetMisc().GetShowRouteToPickup(),
			CancellationPolicy:  s.GetMisc().GetCancellationPolicy(),
			HelpCentreUrl:       s.GetMisc().GetHelpCentreUrl(),
		},
		Prebook: Prebook{
			Enabled:             s.GetPrebook().GetEnabled(),
			ShowDestination:     s.GetPrebook().GetShowDestination(),
			ShowPrice:           s.GetPrebook().GetShowPrice(),
			PollInterval:        s.GetPrebook().GetPollInterval(),
			StartupPollInterval: s.GetPrebook().GetStartupPollInterval(),
			ShowFilters:         s.GetPrebook().GetShowFilters(),
		},
		FastestFirst: s.GetFastestFirst(),
		H4B: H4B{
			RestrictDrivers: s.GetH4B().GetRestrictDrivers(),
		},
		VehicleOptions: VehicleOptions{
			Accessible: s.GetVehicleOptions().GetAccessible(),
			Passengers: s.GetVehicleOptions().GetPassengers(),
		},
		JobOfferVolumeOverride: s.GetJobOfferVolumeOverride(),
		CustomJobRingtone:      s.GetCustomJobRingtone(),
		Fastpay:                fastpayProtoToFastpay(s.GetFastpay()),
	}
}

func fastpayProtoToFastpay(pb *hobpb.Fastpay) Fastpay {
	countriesProto := pb.GetCountries()
	countries := make(map[string][]FastpayCountry)

	for _, country := range countriesProto.GetPreferred() {
		countries["preferred"] = append(countries["preferred"], fastpayCountryProtoToFastpayCountry(country))
	}

	for _, country := range countriesProto.GetOthers() {
		countries["others"] = append(countries["others"], fastpayCountryProtoToFastpayCountry(country))
	}

	return Fastpay{
		Enabled:   pb.GetEnabled(),
		Countries: countries,
	}
}

func fastpayCountryProtoToFastpayCountry(country *hobpb.Fastpay_Country) FastpayCountry {
	return FastpayCountry{
		Name: country.GetName(),
		Code: country.GetCode(),
	}
}
