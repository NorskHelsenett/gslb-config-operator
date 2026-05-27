package records

// all read options currently supported
type ReadOptions struct {
	ZoneName   *string
	ZoneID     *string
	RecordID   *string
	RecordName *string
}

// sets different options of a ReadOptions struct
type ReadOption func(*ReadOptions)

func WithZoneName(zone string) ReadOption {
	return func(ro *ReadOptions) {
		ro.ZoneName = new(zone)
	}
}

func WithZoneID(id string) ReadOption {
	return func(ro *ReadOptions) {
		ro.ZoneID = new(id)
	}
}

func WithRecordID(id string) ReadOption {
	return func(ro *ReadOptions) {
		ro.RecordID = new(id)
	}
}

func WithRecordName(name string) ReadOption {
	return func(ro *ReadOptions) {
		ro.RecordName = new(name)
	}
}
