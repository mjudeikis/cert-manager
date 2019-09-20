/*
Copyright 2019 The Jetstack cert-manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package acmeorders

import (
	"context"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	acmecl "github.com/jetstack/cert-manager/pkg/acme/client"
	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

func TestChallengeSpecForAuthorization(t *testing.T) {
	// a reusable and very simple ACME client that only implements the HTTP01
	// and DNS01 challenge response/record methods
	basicACMEClient := &acmecl.FakeACME{
		FakeHTTP01ChallengeResponse: func(string) (string, error) {
			return "http01", nil
		},
		FakeDNS01ChallengeRecord: func(string) (string, error) {
			return "dns01", nil
		},
	}
	// define some reusable solvers that are used in multiple unit tests
	emptySelectorSolverHTTP01 := v1alpha2.ACMEChallengeSolver{
		HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
			Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
				Name: "empty-selector-solver",
			},
		},
	}
	emptySelectorSolverDNS01 := v1alpha2.ACMEChallengeSolver{
		DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
			Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
				Email: "test-cloudflare-email",
			},
		},
	}
	nonMatchingSelectorSolver := v1alpha2.ACMEChallengeSolver{
		Selector: &v1alpha2.CertificateDNSNameSelector{
			MatchLabels: map[string]string{
				"label":    "does-not-exist",
				"does-not": "match",
			},
		},
		HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
			Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
				Name: "non-matching-selector-solver",
			},
		},
	}
	exampleComDNSNameSelectorSolver := v1alpha2.ACMEChallengeSolver{
		Selector: &v1alpha2.CertificateDNSNameSelector{
			DNSNames: []string{"example.com"},
		},
		HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
			Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
				Name: "example-com-dns-name-selector-solver",
			},
		},
	}
	// define ACME challenges that are used during tests
	acmeChallengeHTTP01 := &v1alpha2.ACMEChallenge{
		Type:  "http-01",
		Token: "http-01-token",
	}
	acmeChallengeDNS01 := &v1alpha2.ACMEChallenge{
		Type:  "dns-01",
		Token: "dns-01-token",
	}

	tests := map[string]struct {
		acmeClient acmecl.Interface
		issuer     v1alpha2.GenericIssuer
		order      *v1alpha2.Order
		authz      *v1alpha2.ACMEAuthorization

		expectedChallengeSpec *v1alpha2.ChallengeSpec
		expectedError         bool
	}{
		"should override the ingress name to edit if override annotation is specified": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{emptySelectorSolverHTTP01},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1alpha2.ACMECertificateHTTP01IngressNameOverride: "test-name-to-override",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "test-name-to-override",
						},
					},
				},
			},
		},
		"should override the ingress class to edit if override annotation is specified": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{emptySelectorSolverHTTP01},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1alpha2.ACMECertificateHTTP01IngressClassOverride: "test-class-to-override",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Class: pointer.StringPtr("test-class-to-override"),
						},
					},
				},
			},
		},
		"should return an error if both ingress class and name override annotations are set": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{emptySelectorSolverHTTP01},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1alpha2.ACMECertificateHTTP01IngressNameOverride:  "test-name-to-override",
						v1alpha2.ACMECertificateHTTP01IngressClassOverride: "test-class-to-override",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedError: true,
		},
		"should ignore HTTP01 override annotations if DNS01 solver is chosen": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{emptySelectorSolverDNS01},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						v1alpha2.ACMECertificateHTTP01IngressNameOverride: "test-name-to-override",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "dns-01",
				DNSName: "example.com",
				Token:   acmeChallengeDNS01.Token,
				Key:     "dns01",
				Solver:  &emptySelectorSolverDNS01,
			},
		},
		"should use configured default solver when no others are present": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{emptySelectorSolverHTTP01},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &emptySelectorSolverHTTP01,
			},
		},
		"should use configured default solver when no others are present but selector is non-nil": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "empty-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "empty-selector-solver",
						},
					},
				},
			},
		},
		"should use configured default solver when others do not match": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverHTTP01,
								nonMatchingSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &emptySelectorSolverHTTP01,
			},
		},
		"should use DNS01 solver over HTTP01 if challenge is of type DNS01": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverHTTP01,
								emptySelectorSolverDNS01,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "dns-01",
				DNSName: "example.com",
				Token:   acmeChallengeDNS01.Token,
				Key:     "dns01",
				Solver:  &emptySelectorSolverDNS01,
			},
		},
		"should return an error if none match": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								nonMatchingSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedError: true,
		},
		"uses correct solver when selector explicitly names dnsName": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverHTTP01,
								exampleComDNSNameSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		"uses default solver if dnsName does not match": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverHTTP01,
								exampleComDNSNameSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"notexample.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "notexample.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "notexample.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &emptySelectorSolverHTTP01,
			},
		},
		"if two solvers specify the same dnsName, the one with the most labels should be chosen": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								exampleComDNSNameSelectorSolver,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label": "exists",
										},
										DNSNames: []string{"example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dns-name-labels-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "exists",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						MatchLabels: map[string]string{
							"label": "exists",
						},
						DNSNames: []string{"example.com"},
					},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "example-com-dns-name-labels-selector-solver",
						},
					},
				},
			},
		},
		"if one solver matches with dnsNames, and the other solver matches with labels, the dnsName solver should be chosen": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								exampleComDNSNameSelectorSolver,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label": "exists",
										},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-labels-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "exists",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		// identical to the test above, but the solvers are listed in reverse
		// order to ensure that this behaviour isn't just incidental
		"if one solver matches with dnsNames, and the other solver matches with labels, the dnsName solver should be chosen (solvers listed in reverse order)": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label": "exists",
										},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-labels-selector-solver",
										},
									},
								},
								exampleComDNSNameSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "exists",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		"if one solver matches with dnsNames, and the other solver matches with 2 labels, the dnsName solver should be chosen": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								exampleComDNSNameSelectorSolver,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label":   "exists",
											"another": "label",
										},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-labels-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label":   "exists",
						"another": "label",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		"should choose the solver with the most labels matching if multiple match": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label": "exists",
										},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-labels-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label":   "exists",
											"another": "matches",
										},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-multiple-labels-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label":   "exists",
						"another": "matches",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						MatchLabels: map[string]string{
							"label":   "exists",
							"another": "matches",
						},
					},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "example-com-multiple-labels-selector-solver",
						},
					},
				},
			},
		},
		"should match wildcard dnsName solver if authorization has Wildcard=true": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverDNS01,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSNames: []string{"*.example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "example-com-wc-dnsname-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"*.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Wildcard:   true,
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:     "dns-01",
				DNSName:  "example.com",
				Wildcard: true,
				Token:    acmeChallengeDNS01.Token,
				Key:      "dns01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSNames: []string{"*.example.com"},
					},
					DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
						Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
							Email: "example-com-wc-dnsname-selector-solver",
						},
					},
				},
			},
		},
		"dnsName selectors should take precedence over dnsZone selectors": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								exampleComDNSNameSelectorSolver,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "com-dnszone-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		"dnsName selectors should take precedence over dnsZone selectors (reversed order)": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "com-dnszone-selector-solver",
										},
									},
								},
								exampleComDNSNameSelectorSolver,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
		"should allow matching with dnsZones": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								emptySelectorSolverDNS01,
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "example-com-dnszone-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.example.com",
				Wildcard:   true,
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:     "dns-01",
				DNSName:  "www.example.com",
				Wildcard: true,
				Token:    acmeChallengeDNS01.Token,
				Key:      "dns01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSZones: []string{"example.com"},
					},
					DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
						Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
							Email: "example-com-dnszone-selector-solver",
						},
					},
				},
			},
		},
		"most specific dnsZone should be selected if multiple match": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "example-com-dnszone-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"prod.example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "prod-example-com-dnszone-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.prod.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.prod.example.com",
				Wildcard:   true,
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:     "dns-01",
				DNSName:  "www.prod.example.com",
				Wildcard: true,
				Token:    acmeChallengeDNS01.Token,
				Key:      "dns01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSZones: []string{"prod.example.com"},
					},
					DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
						Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
							Email: "prod-example-com-dnszone-selector-solver",
						},
					},
				},
			},
		},
		"most specific dnsZone should be selected if multiple match (reversed)": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"prod.example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "prod-example-com-dnszone-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
									},
									DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
										Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
											Email: "example-com-dnszone-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.prod.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.prod.example.com",
				Wildcard:   true,
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeDNS01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:     "dns-01",
				DNSName:  "www.prod.example.com",
				Wildcard: true,
				Token:    acmeChallengeDNS01.Token,
				Key:      "dns01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSZones: []string{"prod.example.com"},
					},
					DNS01: &v1alpha2.ACMEChallengeSolverDNS01{
						Cloudflare: &v1alpha2.ACMEIssuerDNS01ProviderCloudflare{
							Email: "prod-example-com-dnszone-selector-solver",
						},
					},
				},
			},
		},
		"if two solvers specify the same dnsZone, the one with the most labels should be chosen": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnszone-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										MatchLabels: map[string]string{
											"label": "exists",
										},
										DNSZones: []string{"example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnszone-labels-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "exists",
					},
				},
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "www.example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						MatchLabels: map[string]string{
							"label": "exists",
						},
						DNSZones: []string{"example.com"},
					},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "example-com-dnszone-labels-selector-solver",
						},
					},
				},
			},
		},
		"if both solvers match dnsNames, and one also matches dnsZones, choose the one that matches dnsZones": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSNames: []string{"www.example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnsname-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
										DNSNames: []string{"www.example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnsname-dnszone-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "www.example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSZones: []string{"example.com"},
						DNSNames: []string{"www.example.com"},
					},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "example-com-dnsname-dnszone-selector-solver",
						},
					},
				},
			},
		},
		"if both solvers match dnsNames, and one also matches dnsZones, choose the one that matches dnsZones (reversed)": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSZones: []string{"example.com"},
										DNSNames: []string{"www.example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnsname-dnszone-selector-solver",
										},
									},
								},
								{
									Selector: &v1alpha2.CertificateDNSNameSelector{
										DNSNames: []string{"www.example.com"},
									},
									HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
										Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
											Name: "example-com-dnsname-selector-solver",
										},
									},
								},
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"www.example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "www.example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "www.example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver: &v1alpha2.ACMEChallengeSolver{
					Selector: &v1alpha2.CertificateDNSNameSelector{
						DNSZones: []string{"example.com"},
						DNSNames: []string{"www.example.com"},
					},
					HTTP01: &v1alpha2.ACMEChallengeSolverHTTP01{
						Ingress: &v1alpha2.ACMEChallengeSolverHTTP01Ingress{
							Name: "example-com-dnsname-dnszone-selector-solver",
						},
					},
				},
			},
		},
		"uses correct solver when selector explicitly names dnsName (reversed)": {
			acmeClient: basicACMEClient,
			issuer: &v1alpha2.Issuer{
				Spec: v1alpha2.IssuerSpec{
					IssuerConfig: v1alpha2.IssuerConfig{
						ACME: &v1alpha2.ACMEIssuer{
							Solvers: []v1alpha2.ACMEChallengeSolver{
								exampleComDNSNameSelectorSolver,
								emptySelectorSolverHTTP01,
							},
						},
					},
				},
			},
			order: &v1alpha2.Order{
				Spec: v1alpha2.OrderSpec{
					DNSNames: []string{"example.com"},
				},
			},
			authz: &v1alpha2.ACMEAuthorization{
				Identifier: "example.com",
				Challenges: []v1alpha2.ACMEChallenge{*acmeChallengeHTTP01},
			},
			expectedChallengeSpec: &v1alpha2.ChallengeSpec{
				Type:    "http-01",
				DNSName: "example.com",
				Token:   acmeChallengeHTTP01.Token,
				Key:     "http01",
				Solver:  &exampleComDNSNameSelectorSolver,
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cs, err := challengeSpecForAuthorization(ctx, test.acmeClient, test.issuer, test.order, *test.authz)
			if err != nil && !test.expectedError {
				t.Errorf("expected to not get an error, but got: %v", err)
				t.Fail()
			}
			if err == nil && test.expectedError {
				t.Errorf("expected to get an error, but got none")
			}
			if !reflect.DeepEqual(cs, test.expectedChallengeSpec) {
				t.Errorf("returned challenge spec was not as expected: %v", pretty.Diff(test.expectedChallengeSpec, cs))
			}
		})
	}
}
