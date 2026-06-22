package receipt

// reconcileTolerance returns the acceptable gap between the summed items and the
// printed total. OCR drops digits and receipts include deposits/discounts, so we
// allow a small absolute floor plus a relative band.
func reconcileTolerance(total int) int {
	tol := max(total/100, 50)
	return tol
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// reconcile fills ComputedTotal and Reconciled on a result.
func reconcile(r *Result) {
	sum := 0
	for _, it := range r.Items {
		sum += it.Price
	}
	r.ComputedTotal = sum
	r.Reconciled = r.Total > 0 && abs(sum-r.Total) <= reconcileTolerance(r.Total)
}
