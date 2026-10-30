package file

import "fmt"

type Info struct {
	Name        string
	Path        string
	IsDir       bool
	ContainerID string
	DisplayName string
	Size        int64
}

func (i Info) SizeString() string {
	if i.IsDir {
		return "-"
	}

	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case i.Size >= gb:
		return fmt.Sprintf("%.1f GB", float64(i.Size)/gb)
	case i.Size >= mb:
		return fmt.Sprintf("%.1f MB", float64(i.Size)/mb)
	case i.Size >= kb:
		return fmt.Sprintf("%.1f KB", float64(i.Size)/kb)
	default:
		return fmt.Sprintf("%d B", i.Size)
	}
}
