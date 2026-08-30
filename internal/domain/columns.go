package domain

const SystemComponentColumns = `system_component_id, rbd_system_id, parent_id, component_name, component_tag_number, active, vendor, formula_code, distribution_type, failure_rate, running_hours, scale_parameter, shape_parameter, connection_type, connection_to_id, position_x, position_y, source_position, target_position, id_node, reliability_value, active_component, total_component, regresi, mtbf, created_at, updated_at, created_by, updated_by`

const RbdSystemColumns = `rbd_system_id, project_id, drawing_name, system_name, running_hours, reliability_total, formula, created_at, updated_at, created_by, updated_by`

const HierarchyColumns = `hierarchy_id, rbd_system_id, parent_id, level, sub_system_name, formula, formula_code, connection_type, realibility_value, running_hours, position_x, position_y, source_id, target_id`

const EdgeColumns = `id_edge, source_id, target_id`

const FailureEventColumns = `failure_event_id, system_component_id, failure_date, failure_number, running_hours, created_at, updated_at, created_by, updated_by`

const WeibullColumns = `weibull_parameter_id, system_component_id, failure_event_hours, n, freq_f, x, y, created_at, updated_at, created_by, updated_by`

const ExponentialColumns = `exponential_parameter_id, system_component_id, failure_event_hours, n, fregf, f_t_median_rank, r_t, inRt, created_at, updated_at, created_by, updated_by`

const PlotColumns = `reliability_plot_id, system_component_id, time_t, reliability_comp, created_at, updated_at, created_by, updated_by`

const HistoryColumns = `history_id, rbd_system_id, hierarchy_id, hierarchy_name, hierarchy_level, formula_code, formula, calculated_reliability, reliability_lookup, component_details, running_hours, calculation_timestamp, calculated_by`
