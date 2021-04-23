package v1alpha1

import (
	"fmt"

	"github.com/kotalco/kotal/apis/shared"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("IPFS peer validation", func() {
	createCases := []struct {
		Title  string
		Peer   *Peer
		Errors field.ErrorList
	}{
		{
			Title: "Peer #1",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						CPU:      "2",
						CPULimit: "1",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.cpuLimit",
					BadValue: "1",
					Detail:   "must be greater than or equal to cpu 2",
				},
			},
		},
		{
			Title: "Peer #2",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "1Gi",
						MemoryLimit: "1Gi",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.memoryLimit",
					BadValue: "1Gi",
					Detail:   "must be greater than memory 1Gi",
				},
			},
		},
		{
			Title: "Peer #3",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "2Gi",
						MemoryLimit: "1Gi",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.memoryLimit",
					BadValue: "1Gi",
					Detail:   "must be greater than memory 2Gi",
				},
			},
		},
	}

	updateCases := []struct {
		Title   string
		Peer    *Peer
		NewPeer *Peer
		Errors  field.ErrorList
	}{
		{
			Title: "Peer #1",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						CPU:      "1",
						CPULimit: "1",
					},
				},
			},
			NewPeer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						CPU:      "2",
						CPULimit: "1",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.cpuLimit",
					BadValue: "1",
					Detail:   "must be greater than or equal to cpu 2",
				},
			},
		},
		{
			Title: "Peer #2",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "1Gi",
						MemoryLimit: "2Gi",
					},
				},
			},
			NewPeer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "1Gi",
						MemoryLimit: "1Gi",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.memoryLimit",
					BadValue: "1Gi",
					Detail:   "must be greater than memory 1Gi",
				},
			},
		},
		{
			Title: "Peer #3",
			Peer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "1Gi",
						MemoryLimit: "2Gi",
					},
				},
			},
			NewPeer: &Peer{
				Spec: PeerSpec{
					Resources: shared.Resources{
						Memory:      "2Gi",
						MemoryLimit: "1Gi",
					},
				},
			},
			Errors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.resources.memoryLimit",
					BadValue: "1Gi",
					Detail:   "must be greater than memory 2Gi",
				},
			},
		},
	}

	Context("While creating peer", func() {
		for _, c := range createCases {
			func() {
				cc := c
				It(fmt.Sprintf("Should validate %s", cc.Title), func() {
					cc.Peer.Default()
					err := cc.Peer.ValidateCreate()

					// all test cases has validation errors
					Expect(err).NotTo(BeNil())

					errStatus := err.(*errors.StatusError)

					causes := shared.ErrorsToCauses(cc.Errors)

					Expect(errStatus.ErrStatus.Details.Causes).To(ContainElements(causes))
				})
			}()
		}
	})

	Context("While updating peer", func() {
		for _, c := range updateCases {
			func() {
				cc := c
				It(fmt.Sprintf("Should validate %s", cc.Title), func() {
					cc.Peer.Default()
					err := cc.NewPeer.ValidateUpdate(cc.Peer)

					// all test cases has validation errors
					Expect(err).NotTo(BeNil())

					errStatus := err.(*errors.StatusError)

					causes := shared.ErrorsToCauses(cc.Errors)

					Expect(errStatus.ErrStatus.Details.Causes).To(ContainElements(causes))
				})
			}()
		}
	})
})