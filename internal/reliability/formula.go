package reliability

import (
	"regexp"
	"sort"
	"strings"
)

type Edge struct {
	SourceId string
	TargetId string
}

type graph struct {
	adjacency *orderedMap[[]string]
	reverse   *orderedMap[[]string]
	incoming  *orderedMap[int]
}

var (
	stepToken   = regexp.MustCompile(`step_\d+`)
	stepPattern = regexp.MustCompile(`\b(Paralel|Seri)\(([^()]+)\)`)
	codeToken   = regexp.MustCompile(`\b[A-Z]+-?\d+\b`)
)

func buildGraph(edges []Edge) graph {
	g := graph{adjacency: newOrderedMap[[]string](), reverse: newOrderedMap[[]string](), incoming: newOrderedMap[int]()}
	for _, edge := range edges {
		if !g.adjacency.Has(edge.SourceId) {
			g.adjacency.Set(edge.SourceId, []string{})
		}
		if !g.reverse.Has(edge.TargetId) {
			g.reverse.Set(edge.TargetId, []string{})
		}
		g.adjacency.Set(edge.SourceId, append(g.adjacency.Get(edge.SourceId), edge.TargetId))
		g.reverse.Set(edge.TargetId, append(g.reverse.Get(edge.TargetId), edge.SourceId))
		if !g.incoming.Has(edge.TargetId) {
			g.incoming.Set(edge.TargetId, 0)
		}
		if !g.incoming.Has(edge.SourceId) {
			g.incoming.Set(edge.SourceId, 0)
		}
		g.incoming.Set(edge.TargetId, g.incoming.Get(edge.TargetId)+1)
	}
	return g
}

func (g graph) children(node string) []string {
	if g.adjacency.Has(node) {
		return g.adjacency.Get(node)
	}
	return []string{}
}

func (g graph) parents(node string) []string {
	if g.reverse.Has(node) {
		return g.reverse.Get(node)
	}
	return []string{}
}

func sorted(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, value := range a {
		set[value] = true
	}
	for _, value := range b {
		if !set[value] {
			return false
		}
	}
	return len(set) == len(a) || sameDistinct(a, b)
}

func sameDistinct(a, b []string) bool {
	left, right := map[string]bool{}, map[string]bool{}
	for _, value := range a {
		left[value] = true
	}
	for _, value := range b {
		right[value] = true
	}
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if !right[value] {
			return false
		}
	}
	return true
}

func expandSteps(formula string) string {
	steps := processSteps(formula)
	result := steps.Get("final_step")
	for range steps.Keys() {
		result = stepToken.ReplaceAllStringFunc(result, func(token string) string {
			if steps.Has(token) {
				return "(" + steps.Get(token) + ")"
			}
			return token
		})
	}
	return result
}

func tryTwoStageParallel(g graph) string {
	roots := []string{}
	for _, key := range g.incoming.Keys() {
		if g.incoming.Get(key) == 0 {
			roots = append(roots, key)
		}
	}
	if len(roots) < 2 {
		return ""
	}
	leaves := []string{}
	for _, key := range g.incoming.Keys() {
		if len(g.children(key)) == 0 {
			leaves = append(leaves, key)
		}
	}
	if len(leaves) < 2 {
		return ""
	}
	sortedLeaves := sorted(leaves)
	for _, root := range roots {
		if !g.adjacency.Has(root) {
			return ""
		}
		if !equalSlices(sorted(g.adjacency.Get(root)), sortedLeaves) {
			return ""
		}
	}
	for _, leaf := range leaves {
		if !g.reverse.Has(leaf) || len(g.reverse.Get(leaf)) != len(roots) {
			return ""
		}
	}
	stage1 := "Paralel(" + strings.Join(sorted(roots), ", ") + ")"
	stage2 := "Paralel(" + strings.Join(sortedLeaves, ", ") + ")"
	return expandSteps("Seri(" + stage1 + ", " + stage2 + ")")
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func tryMultiStageParallel(g graph) string {
	leafNodes := []string{}
	for _, key := range g.incoming.Keys() {
		if len(g.children(key)) == 0 {
			leafNodes = append(leafNodes, key)
		}
	}
	if len(leafNodes) < 2 {
		return ""
	}
	firstPreds := g.parents(leafNodes[0])
	for _, leaf := range leafNodes[1:] {
		if !sameDistinct(g.parents(leaf), firstPreds) {
			return ""
		}
	}
	leafPreds := sorted(distinct(firstPreds))
	if len(leafPreds) == 0 {
		return ""
	}
	for _, pred := range leafPreds {
		successors := map[string]bool{}
		for _, successor := range g.children(pred) {
			successors[successor] = true
		}
		for _, leaf := range leafNodes {
			if !successors[leaf] {
				return ""
			}
		}
	}
	parallelStages := 0
	for _, node := range g.adjacency.Keys() {
		if len(g.adjacency.Get(node)) > 1 {
			parallelStages++
		}
	}
	if parallelStages < 2 {
		return ""
	}
	leafFormula := "Paralel(" + strings.Join(sorted(leafNodes), ", ") + ")"
	predecessorFormula := processPredecessorLayers(leafPreds, g)
	return "Seri(" + predecessorFormula + ", " + leafFormula + ")"
}

func distinct(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func processPredecessorLayers(nodes []string, g graph) string {
	if len(nodes) == 0 {
		return ""
	}
	if len(nodes) == 1 {
		single := nodes[0]
		preds := g.parents(single)
		if len(preds) == 1 {
			predFormula := processPredecessorLayers(preds, g)
			if predFormula == "" {
				return single
			}
			return "Seri(" + predFormula + ", " + single + ")"
		}
		if len(preds) > 1 {
			predFormula := processPredecessorLayers(sorted(preds), g)
			if predFormula == "" {
				return single
			}
			return "Seri(" + predFormula + ", " + single + ")"
		}
		return single
	}
	firstPreds := g.parents(nodes[0])
	allSamePreds := true
	for _, node := range nodes[1:] {
		if !sameDistinct(g.parents(node), firstPreds) {
			allSamePreds = false
			break
		}
	}
	firstSuccs := g.children(nodes[0])
	allSameSuccs := true
	for _, node := range nodes[1:] {
		if !sameDistinct(g.children(node), firstSuccs) {
			allSameSuccs = false
			break
		}
	}
	if allSamePreds && allSameSuccs && len(nodes) > 1 {
		parallel := "Paralel(" + strings.Join(sorted(nodes), ", ") + ")"
		if len(distinct(firstPreds)) == 0 {
			return parallel
		}
		parent := processPredecessorLayers(sorted(distinct(firstPreds)), g)
		return "Seri(" + parent + ", " + parallel + ")"
	}
	return "Paralel(" + strings.Join(sorted(nodes), ", ") + ")"
}

func GenerateFormula(edges []Edge) string {
	g := buildGraph(edges)
	if twoStage := tryTwoStageParallel(g); twoStage != "" {
		return twoStage
	}
	if multiStage := tryMultiStageParallel(g); multiStage != "" {
		return expandSteps(multiStage)
	}
	queueKeys := g.incoming.Keys()
	sort.SliceStable(queueKeys, func(i, j int) bool { return g.incoming.Get(queueKeys[i]) < g.incoming.Get(queueKeys[j]) })
	queue := queueKeys
	result := newOrderedMap[string]()
	deletedIncoming := newOrderedMap[int]()
	deletedChildren := newOrderedMap[int]()
	deletedPara := map[string]bool{}
	parInSeri := 0
	subKey := []string{}
	roots := []string{}
	for _, key := range g.incoming.Keys() {
		if g.incoming.Get(key) == 0 {
			roots = append(roots, key)
		}
	}
	paraClose := ""
	resolved := func(node string) string {
		if result.Has(node) && result.Get(node) != "" {
			return result.Get(node)
		}
		return node
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children := g.children(current)
		if len(children) > 1 {
			childResults := make([]string, 0, len(children))
			for _, child := range children {
				childResults = append(childResults, resolved(child))
			}
			grandSon := g.children(childResults[0])
			if len(grandSon) > 0 {
				switch {
				case g.reverse.Has(grandSon[0]) && len(g.reverse.Get(grandSon[0])) == 1:
					result.Set(current, current+", Paralel(")
					paraClose = children[len(children)-1]
				case g.adjacency.Has(current) && len(g.adjacency.Get(current)) == 1:
					result.Set(current, current+", Paralel(")
					paraClose = children[len(children)-1]
				case g.reverse.Has(grandSon[0]) && len(g.reverse.Get(grandSon[0])) > 1 && len(children) > 1:
					result.Set(current, "Seri("+current+", Paralel("+strings.Join(childResults, ", ")+"), "+grandSon[0]+")")
					if len(g.children(grandSon[0])) == 0 {
						deletedChildren.Set(grandSon[0], 1)
					} else {
						deletedPara[grandSon[0]] = true
					}
				default:
					result.Set(current, current+", Paralel("+strings.Join(childResults, ", ")+")")
				}
			} else {
				switch {
				case len(roots) > 1:
					result.Set(current, current+", Paralel("+strings.Join(childResults, ", ")+")")
				case deletedPara[current]:
					result.Set(current, "Paralel("+strings.Join(childResults, ", ")+")")
				default:
					result.Set(current, "), "+current+", Paralel("+strings.Join(childResults, ", ")+")")
				}
			}
			if len(queue) == 0 {
				for _, child := range children {
					deletedChildren.Set(child, 1)
				}
			}
		} else if len(children) == 1 {
			child := children[0]
			if g.reverse.Has(child) && len(g.reverse.Get(child)) == 1 {
				lewatPara := ""
				if len(queue) == 0 {
					result.Set(current, current)
				} else {
					childText := child
					if result.Has(child) {
						childText = result.Get(child)
					}
					result.Set(current, "Seri("+current+", "+childText)
				}
				remaining := len(queue)
				for step := 0; step < remaining; step++ {
					grandSon := g.children(child)
					if len(grandSon) == 1 {
						gs := grandSon[0]
						gsText := gs
						if result.Has(gs) {
							gsText = result.Get(gs)
						}
						if g.reverse.Has(gs) && len(g.reverse.Get(gs)) == 1 {
							result.Set(current, result.Get(current)+", "+gsText)
							if g.incoming.Has(child) {
								deletedChildren.Set(child, g.incoming.Get(child))
							}
							child = gs
						} else if g.reverse.Has(gs) && len(g.reverse.Get(gs)) > 1 && parInSeri > 0 {
							result.Set(current, result.Get(current)+", "+gsText)
							if g.incoming.Has(child) {
								deletedChildren.Set(child, g.incoming.Get(child))
							}
							child = gs
							parInSeri--
						} else if parInSeri > 0 {
							child = gs
						} else {
							break
						}
					} else if len(grandSon) > 1 {
						gs := grandSon[0]
						gsChild := g.children(gs)
						if len(gsChild) > 0 && g.reverse.Has(gsChild[0]) && len(g.reverse.Get(gsChild[0])) > 1 {
							result.Set(current, result.Get(current)+", Paralel("+strings.Join(grandSon, ", ")+")")
							if g.incoming.Has(child) {
								deletedChildren.Set(child, g.incoming.Get(child))
							}
							child = gs
							parInSeri++
						} else {
							result.Set(current, result.Get(current)+", Paralel("+strings.Join(grandSon, ", ")+")")
							if g.incoming.Has(child) {
								deletedChildren.Set(child, g.incoming.Get(child))
							}
							child = gs
							subKey = append(subKey, grandSon...)
							break
						}
					} else {
						if !g.adjacency.Has(child) {
							if g.incoming.Has(child) {
								deletedChildren.Set(child, g.incoming.Get(child))
							}
						}
						break
					}
				}
				if current == paraClose {
					result.Set(current, result.Get(current)+")")
					paraClose = ""
					lewatPara = "lewat"
				}
				if len(queue) > 0 && (lewatPara == "" || !g.adjacency.Has(child)) {
					result.Set(current, result.Get(current)+")")
				}
			} else {
				if g.incoming.Has(current) && g.incoming.Get(current) == 0 {
					result.Set(current, current)
				} else {
					result.Set(current, "")
				}
			}
		} else {
			if g.reverse.Has(current) {
				if len(g.reverse.Get(current)) > 1 && len(roots) <= 1 {
					result.Set(current, "), "+current)
				} else {
					for _, parent := range g.reverse.Get(current) {
						if g.adjacency.Has(parent) && len(g.adjacency.Get(parent)) == 1 {
							result.Set(current, current)
						} else {
							result.Set(current, "")
						}
					}
				}
			}
		}
		if g.incoming.Has(current) {
			deletedIncoming.Set(current, g.incoming.Get(current))
			g.incoming.Remove(current)
		} else {
			deletedIncoming.Set(current, 0)
		}
	}
	if len(subKey) > 0 {
		for _, sk := range subKey {
			for _, key := range result.Keys() {
				value := result.Get(key)
				if strings.Contains(value, sk) {
					newValue := result.Get(sk)
					if newValue != "" {
						result.Set(key, strings.ReplaceAll(value, sk, newValue))
						result.Set(sk, "")
					}
					break
				}
			}
		}
	}
	deletedRoots := map[string]bool{}
	for _, key := range deletedIncoming.Keys() {
		if deletedIncoming.Get(key) == 0 {
			deletedRoots[key] = true
		}
	}
	for _, key := range deletedChildren.Keys() {
		deletedRoots[key] = true
	}
	finalResult := ""
	if len(roots) > 1 {
		parts := make([]string, 0, len(roots))
		for _, root := range roots {
			if result.Has(root) {
				parts = append(parts, result.Get(root))
			} else {
				parts = append(parts, root)
			}
		}
		finalResult = "Paralel(" + strings.Join(parts, ", ") + ")"
	} else if len(roots) == 1 {
		root := roots[0]
		if result.Has(root) {
			finalResult = result.Get(root)
		} else {
			finalResult = root
		}
	}
	for _, node := range deletedIncoming.Keys() {
		if deletedRoots[node] {
			continue
		}
		if result.Has(node) && result.Get(node) != "" {
			finalResult = finalResult + ", " + result.Get(node)
		}
	}
	finalResult = strings.TrimLeft(finalResult, ") ,")
	finalResult = removeDuplicates(finalResult)
	return expandSteps(strings.ReplaceAll(strings.ReplaceAll(finalResult, "(, ", "("), ", )", ")"))
}

func removeDuplicates(input string) string {
	seen := map[string]bool{}
	result := codeToken.ReplaceAllStringFunc(input, func(match string) string {
		if seen[match] {
			return ""
		}
		seen[match] = true
		return match
	})
	result = strings.ReplaceAll(result, ", ,", ",")
	result = strings.ReplaceAll(result, "(,", "(")
	result = strings.ReplaceAll(result, ",)", ")")
	for strings.HasSuffix(result, ",") || strings.HasSuffix(result, "*") || strings.HasSuffix(result, "+") || strings.HasSuffix(result, "-") {
		result = result[:len(result)-1]
	}
	return strings.TrimSpace(result)
}

func processSteps(input string) *orderedMap[string] {
	stepCounter := 1
	steps := newOrderedMap[string]()
	expression := input
	for {
		location := stepPattern.FindStringSubmatchIndex(expression)
		if location == nil {
			break
		}
		matched := expression[location[0]:location[1]]
		functionType := expression[location[2]:location[3]]
		inner := expression[location[4]:location[5]]
		components := []string{}
		for _, part := range strings.Split(inner, ",") {
			components = append(components, strings.TrimSpace(part))
		}
		formula := ""
		if functionType == "Seri" {
			formula = strings.Join(components, "*")
		} else {
			wrapped := make([]string, 0, len(components))
			for _, component := range components {
				wrapped = append(wrapped, "(1-"+component+")")
			}
			formula = "1-" + strings.Join(wrapped, "*")
		}
		placeholder := "step_" + itoa(stepCounter)
		steps.TryAdd(placeholder, formula)
		stepCounter++
		expression = strings.ReplaceAll(expression, matched, placeholder)
	}
	components := []string{}
	for _, part := range strings.Split(expression, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			components = append(components, trimmed)
		}
	}
	steps.TryAdd("final_step", strings.Join(components, "*"))
	return steps
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
