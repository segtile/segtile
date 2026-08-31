package superpixel_utils

import (
	"image"
	"sort"
)

// cross returns the z-component of the cross product of vectors o->a and o->b.
func cross(o, a, b image.Point) int {
	return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
}

// ConvexHull computes the convex hull of a set of points using the monotone chain algorithm.
func ConvexHull(points []image.Point) []image.Point {
	if len(points) < 3 {
		return points
	}

	sort.Slice(points, func(i, j int) bool {
		if points[i].X != points[j].X {
			return points[i].X < points[j].X
		}
		return points[i].Y < points[j].Y
	})

	build := func(pts []image.Point) []image.Point {
		hull := make([]image.Point, 0, len(pts))
		for _, p := range pts {
			for len(hull) >= 2 && cross(hull[len(hull)-2], hull[len(hull)-1], p) <= 0 {
				hull = hull[:len(hull)-1]
			}
			hull = append(hull, p)
		}
		return hull
	}

	lower := build(points)

	reversed := make([]image.Point, len(points))
	for i, p := range points {
		reversed[len(points)-1-i] = p
	}
	upper := build(reversed)

	lower = lower[:len(lower)-1]
	upper = upper[:len(upper)-1]
	return append(lower, upper...)
}
