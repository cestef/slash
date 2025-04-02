package license

import (
	"context"
	_ "embed"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/yourselfhosted/slash/proto/gen/api/v1"
	storepb "github.com/yourselfhosted/slash/proto/gen/store"
	"github.com/yourselfhosted/slash/server/profile"
	"github.com/yourselfhosted/slash/store"
)

type LicenseService struct {
	Profile *profile.Profile
	Store   *store.Store

	cachedSubscription *v1pb.Subscription
}

// NewLicenseService creates a new LicenseService.
func NewLicenseService(profile *profile.Profile, store *store.Store) *LicenseService {
	return &LicenseService{
		Profile:            profile,
		Store:              store,
		cachedSubscription: getSubscriptionForFreePlan(),
	}
}

func (s *LicenseService) LoadSubscription(ctx context.Context) (*v1pb.Subscription, error) {
	workspaceGeneralSetting, err := s.Store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace general setting")
	}

	subscription := getSubscriptionForFreePlan()
	licenseKey := workspaceGeneralSetting.LicenseKey
	if licenseKey == "" {
		s.cachedSubscription = subscription
		return subscription, nil
	}

	result, err := validateLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate license key")
	}
	if result == nil {
		s.cachedSubscription = subscription
		return subscription, nil
	}

	subscription.Plan = result.Plan
	subscription.ExpiresTime = timestamppb.New(result.ExpiresTime)
	subscription.Seats = int32(result.Seats)
	for _, feature := range result.Features {
		subscription.Features = append(subscription.Features, feature.String())
	}
	s.cachedSubscription = subscription
	return subscription, nil
}

func (s *LicenseService) UpdateSubscription(ctx context.Context, licenseKey string) (*v1pb.Subscription, error) {
	if licenseKey != "" {
		result, err := validateLicenseKey(licenseKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate license key")
		}
		if result == nil {
			return nil, errors.New("invalid license key")
		}
	}
	if err := s.UpdateLicenseKey(ctx, licenseKey); err != nil {
		return nil, errors.Wrap(err, "failed to update license key")
	}

	subscription, err := s.LoadSubscription(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load subscription")
	}
	return subscription, nil
}

func (s *LicenseService) UpdateLicenseKey(ctx context.Context, licenseKey string) error {
	workspaceGeneralSetting, err := s.Store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace general setting")
	}
	workspaceGeneralSetting.LicenseKey = licenseKey
	_, err = s.Store.UpsertWorkspaceSetting(ctx, &storepb.WorkspaceSetting{
		Key: storepb.WorkspaceSettingKey_WORKSPACE_SETTING_GENERAL,
		Value: &storepb.WorkspaceSetting_General{
			General: workspaceGeneralSetting,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to upsert workspace setting")
	}
	return nil
}

func (s *LicenseService) GetSubscription() *v1pb.Subscription {
	return s.cachedSubscription
}

func (s *LicenseService) IsFeatureEnabled(feature FeatureType) bool {
	// return slices.Contains(s.cachedSubscription.Features, feature.String())
	return true
}

type ValidateResult struct {
	Plan        v1pb.PlanType
	ExpiresTime time.Time
	Trial       bool
	Seats       int
	Features    []FeatureType
}

type Claims struct {
	jwt.RegisteredClaims

	Owner string `json:"owner"`
	Plan  string `json:"plan"`
	Trial bool   `json:"trial"`
	// The number of seats in the license key. Leave it empty if the license key does not have a seat limit.
	Seats int `json:"seats"`
	// The available features in the license key.
	Features []string `json:"features"`
}

func validateLicenseKey(licenseKey string) (*ValidateResult, error) {
	// Try to parse the license key as a JWT token.
	claims, _ := parseLicenseKey(licenseKey)
	if claims != nil {
		result := &ValidateResult{
			Plan:        v1pb.PlanType(v1pb.PlanType_value[claims.Plan]),
			ExpiresTime: claims.ExpiresAt.Time,
			Trial:       claims.Trial,
			Seats:       claims.Seats,
		}
		result.Features = getDefaultFeatures(result.Plan)

		plan := v1pb.PlanType(v1pb.PlanType_value[claims.Plan])
		if plan == v1pb.PlanType_PLAN_TYPE_UNSPECIFIED {
			return nil, errors.New("invalid plan")
		}
		return result, nil
	}

	// Otherwise, return an error.
	return nil, errors.New("invalid license key")
}

func parseLicenseKey(licenseKey string) (*Claims, error) {
	inAHundredYears := time.Now().Add(time.Hour * 24 * 365 * 100)
	return &Claims{
		Owner: "me",
		Plan:  "PRO",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(inAHundredYears),
		},
		Trial: false,
		Seats: 0xDEAD,
		Features: []string{
			"no_license",
			"why_would_you_need_a_license",
		},
	}, nil
}

func getSubscriptionForFreePlan() *v1pb.Subscription {
	return &v1pb.Subscription{
		Plan:             v1pb.PlanType_PRO,
		Seats:            0xC0FFEE, // Corrected hexadecimal value
		CollectionsLimit: 0xDEAD,
		ShortcutsLimit:   0xBEEF,
		Features: []string{
			"no_license",
			"why_would_you_need_a_license",
		},
	}
}
