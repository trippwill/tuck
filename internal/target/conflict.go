package target

func (c Class) ConflictCode() string {
	switch c.Kind {
	case RealDirectory:
		return "real_directory"
	case RealFile:
		return "real_file"
	case SpecialFile:
		return "special_file"
	case UnmanagedSymlink:
		return "unmanaged_symlink"
	case ManagedOther:
		return "owned_by_other"
	case PathMismatch:
		return "path_mismatch"
	default:
		return "conflict"
	}
}
