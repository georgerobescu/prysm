package validators

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestationParticipants_ok(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	attestationData := &pb.AttestationData{}

	tests := []struct {
		attestationSlot uint64
		stateSlot       uint64
		shard           uint64
		bitfield        []byte
		wanted          []uint64
	}{
		{
			attestationSlot: 2,
			stateSlot:       5,
			shard:           2,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{11, 121},
		},
		{
			attestationSlot: 1,
			stateSlot:       10,
			shard:           1,
			bitfield:        []byte{77},
			wanted:          []uint64{117},
		},
		{
			attestationSlot: 10,
			stateSlot:       20,
			shard:           10,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{14, 30},
		},
		{
			attestationSlot: 64,
			stateSlot:       100,
			shard:           0,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{109, 97},
		},
		{
			attestationSlot: 999,
			stateSlot:       1000,
			shard:           39,
			bitfield:        []byte{99},
			wanted:          []uint64{89},
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot
		attestationData.Slot = tt.attestationSlot
		attestationData.Shard = tt.shard

		result, err := AttestationParticipants(state, attestationData, tt.bitfield)
		if err != nil {
			t.Errorf("Failed to get attestation participants: %v", err)
		}

		if !reflect.DeepEqual(tt.wanted, result) {
			t.Errorf(
				"Result indices was an unexpected value. Wanted %d, got %d",
				tt.wanted,
				result,
			)
		}
	}
}

func TestAttestationParticipants_IncorrectBitfield(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	validators := make([]*pb.ValidatorRecord, validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{1}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}

func TestCommitteeCountPerSlot_Ok(t *testing.T) {
	// this defines the # of validators required to have 1 committee
	// per slot for epoch length.
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		committeeCount uint64
	}{
		{0, 1},
		{1000, 1},
		{2 * validatorsPerEpoch, 2},
		{5 * validatorsPerEpoch, 5},
		{16 * validatorsPerEpoch, 16},
		{32 * validatorsPerEpoch, 16},
	}
	for _, test := range tests {
		if test.committeeCount != committeeCountPerSlot(test.validatorCount) {
			t.Errorf("wanted: %d, got: %d",
				test.committeeCount, committeeCountPerSlot(test.validatorCount))
		}
	}
}

func TestCurrCommitteesCountPerSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(8)
	// set curr epoch total validators count to 8 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if CurrCommitteesCountPerSlot(state) != committeesPerEpoch {
		t.Errorf("Incorrect current epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, CurrCommitteesCountPerSlot(state))
	}
}

func TestPrevCommitteesCountPerSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(3)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if prevCommitteesCountPerSlot(state) != committeesPerEpoch {
		t.Errorf("Incorrect prev epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, prevCommitteesCountPerSlot(state))
	}
}

func TestShuffling_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	randaoSeed := [32]byte{'A'}
	slot := uint64(10)
	committees, err := Shuffling(randaoSeed, validators, slot)
	if err != nil {
		t.Fatalf("Could not shuffle validators: %v", err)
	}

	// Verify shuffled list is correctly split into
	// epoch_length * committees_per_slot pieces.
	committeesPerSlot := committeeCountPerSlot(uint64(len(validators)))
	if len(committees) != int(committeesPerSlot*config.EpochLength) {
		t.Errorf("Incorrect committee count after splitting. Wanted: %d, got: %d",
			committeesPerSlot*config.EpochLength, len(committees))
	}

	// Verify each shuffled committee is TARGET_COMMITTEE_SIZE.
	for i := 0; i < len(committees); i++ {
		if len(committees[i]) != int(config.TargetCommitteeSize) {
			t.Errorf("Incorrect validator count per committee. Wanted: %d, got: %d",
				config.TargetCommitteeSize, len(committees[i]))
		}
	}

}

func TestShuffling_OutOfBound(t *testing.T) {
	populateValidatorsMax()
	if _, err := Shuffling([32]byte{}, validatorsUpperBound, 0); err == nil {
		t.Fatalf("Shuffling should have failed with exceeded upper bound")
	}
}

func TestCrosslinkCommitteesAtSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              200,
	}
	committees, err := CrosslinkCommitteesAtSlot(state, 132)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}
	if len(committees) != int(committeesPerEpoch) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, len(committees))
	}

	newCommittees, err := CrosslinkCommitteesAtSlot(state, 180)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}

	if reflect.DeepEqual(committees, newCommittees) {
		t.Error("Committees from different slot shall not be equal")
	}
}

func TestCrosslinkCommitteesAtSlot_OutOfBound(t *testing.T) {
	want := fmt.Sprintf(
		"input committee slot %d out of bounds: %d <= slot < %d",
		config.EpochLength+1, 0, config.EpochLength,
	)

	if _, err := CrosslinkCommitteesAtSlot(&pb.BeaconState{}, config.EpochLength+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestCrosslinkCommitteesAtPrevSlot_ShuffleFailed(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: validatorsUpperBound,
		Slot:              100,
	}

	want := fmt.Sprint(
		"could not shuffle prev epoch validators: " +
			"input list exceeded upper bound and reached modulo bias",
	)

	if _, err := CrosslinkCommitteesAtSlot(state, 1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestCrosslinkCommitteesAtCurrSlot_ShuffleFailed(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: validatorsUpperBound,
		Slot:              100,
	}

	want := fmt.Sprint(
		"could not shuffle current epoch validators: " +
			"input list exceeded upper bound and reached modulo bias",
	)

	if _, err := CrosslinkCommitteesAtSlot(state, 99); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
