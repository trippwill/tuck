package target

type ConflictCode string

const (
	ConflictAbsent            ConflictCode = "absent"
	ConflictGeneric           ConflictCode = "conflict"
	ConflictInsideSourceRepo  ConflictCode = "inside_source_repo"
	ConflictMultipleProviders ConflictCode = "multiple_providers"
	ConflictNotManagedSymlink ConflictCode = "not_a_managed_symlink"
	ConflictOutsideTargetRoot ConflictCode = "outside_target_root"
	ConflictPackagePathExists ConflictCode = "package_path_exists"
	ConflictPathMismatch      ConflictCode = "path_mismatch"
	ConflictRealDirectory     ConflictCode = "real_directory"
	ConflictRealFile          ConflictCode = "real_file"
	ConflictSpecialFile       ConflictCode = "special_file"
	ConflictUnmanagedSymlink  ConflictCode = "unmanaged_symlink"
	ConflictOwnedByOther      ConflictCode = "owned_by_other"
	ConflictCopySourceChanged ConflictCode = "copy_source_modified"
	ConflictCopyTargetChanged ConflictCode = "copy_target_modified"
	ConflictCopyDrift         ConflictCode = "copy_drift"
)

func (c Class) ConflictCode() ConflictCode {
	switch c.Kind {
	case RealDirectory:
		return ConflictRealDirectory
	case RealFile:
		return ConflictRealFile
	case SpecialFile:
		return ConflictSpecialFile
	case UnmanagedSymlink:
		return ConflictUnmanagedSymlink
	case ManagedOther:
		return ConflictOwnedByOther
	case PathMismatch:
		return ConflictPathMismatch
	default:
		return ConflictGeneric
	}
}
