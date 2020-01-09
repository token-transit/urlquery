package urlquery

type builderOptions struct {
	urlEncoder       UrlEncoder
	ignoreEmptyValue bool
}

type BuilderOption interface {
	apply(*builderOptions)
}

type urlEncoderOption struct {
	urlEncoder UrlEncoder
}

func (o urlEncoderOption) apply(opts *builderOptions) {
	opts.urlEncoder = o.urlEncoder
}

//support customized urlEncoder option
func WithUrlEncoderOption(u UrlEncoder) BuilderOption {
	return urlEncoderOption{urlEncoder: u}
}

type ignoreEmptyValueOption bool

func (o ignoreEmptyValueOption) apply(opts *builderOptions) {
	opts.ignoreEmptyValue = bool(o)
}

//support to control whether to ignore empty value.
//It just happen to the element directly in strcut, not include map slice array
//default:false, meaning not to ignore
func WithIgnoreEmptyValueOption(c bool) BuilderOption {
	return ignoreEmptyValueOption(c)
}