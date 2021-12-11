package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type day_start_end struct {
	day          string
	start_hour   int
	start_minute int
	end_hour     int
	end_minute   int
}

type day_start_end_date_minutes struct {
	day        string
	start_date time.Time
	end_date   time.Time
	minutes    float64
}

type Account_value_quantity_barcode struct {
	Account  string
	value    float64
	quantity float64
	barcode  string
}

type account_method_value_price struct {
	account, method         string
	value_or_percent, price float64
}

type invoice_struct struct {
	account                string
	value, price, quantity float64
}

type account struct {
	status, father, name string
}

type Financial_accounting struct {
	date_layout []string
	DriverName, DataSourceName, Database_name, invoice_discount, retained_earnings,
	income_statement, assets, current_assets, current_liabilities string
	accounts               []account
	Invoice_discounts_list [][2]float64
	auto_complete_entries  [][]account_method_value_price
}

type journal_tag struct {
	date          string
	entry_number  int
	account       string
	value         float64
	price         float64
	quantity      float64
	barcode       string
	entry_expair  string
	description   string
	name          string
	employee_name string
	entry_date    string
	reverse       bool
}

type statement struct {
	account string
	value, price, quantity, percent,
	inflow, outflow, flow, flow_percent,
	change_in_amount_since_base_period, current_results_in_relation_to_base_period float64
}

type financial_analysis struct {
	current_assets, current_liabilities, cash, short_term_investments, net_receivables, net_credit_sales,
	average_net_receivables, cost_of_goods_sold, average_inventory, net_income, net_sales, average_assets,
	preferred_dividends, average_common_stockholders_equity, market_price_per_shares_outstanding, cash_dividends,
	total_debt, total_assets, ebitda, interest_expense, weighted_average_common_shares_outstanding float64
}

type financial_analysis_statement struct {
	current_ratio                        float64 // current_assets / current_liabilities
	acid_test                            float64 // (cash + short_term_investments + net_receivables) / current_liabilities
	receivables_turnover                 float64 // net_credit_sales / average_net_receivables
	inventory_turnover                   float64 // cost_of_goods_sold / average_inventory
	profit_margin                        float64 // net_income / net_sales
	asset_turnover                       float64 // net_sales / average_assets
	return_on_assets                     float64 // net_income / average_assets
	return_on_common_stockholders_equity float64 // (net_income - preferred_dividends) / average_common_stockholders_equity
	earnings_per_share                   float64 // (net_income - preferred_dividends) / weighted_average_common_shares_outstanding
	price_earnings_ratio                 float64 // market_price_per_shares_outstanding / earnings_per_share
	payout_ratio                         float64 // cash_dividends / net_income
	debt_to_total_assets_ratio           float64 // total_debt / total_assets
	times_interest_earned                float64 // ebitda / interest_expense
}

type value_quantity struct {
	value, quantity, inflow, outflow float64
}

type cvp_statistics struct {
	name string
	units,
	selling_price_per_unit,
	variable_cost_per_unit,
	fixed_cost,
	mixed_cost,
	mixed_cost_per_unit,
	sales,
	profit,
	profit_per_unit,
	contribution_margin_per_unit,
	contribution_margin,
	contribution_margin_ratio,
	break_even_in_unit,
	break_even_in_sales,
	degree_of_operating_leverage float64
}

type cvp struct {
	name                                                              string
	units, selling_price_per_unit, variable_cost_per_unit, fixed_cost float64
	portions                                                          []float64
}

type overhead struct {
	percent_method string
	fixed_cost     []float64
}

type Managerial_Accounting struct {
	points_activity_level_and_cost_at_the_activity_level [][2]float64
	cvp                                                  []cvp
	overhead                                             []overhead
	beginning_balance,
	increase,
	ending_balance,
	decreases_in_account_caused_by_not_sell,
	actual_mixed_cost,
	number_of_partially_completed_units,
	percentage_completion,
	units_transferred_to_the_next_department_or_to_finished_goods,
	equivalent_units_in_ending_work_in_process_inventory,
	equivalent_units_in_beginning_work_in_process_inventory,
	cost_of_beginning_work_in_process_inventory,
	cost_added_during_the_period float64
}

var (
	inventory, current_assets, assets_normal, assets_contra, liabilities_normal, equity_normal,
	revenues, expenses, temporary_debit_accounts, temporary_accounts, debit_accounts, credit_accounts,
	cash_and_cash_equivalent, fifo, lifo, wma, short_term_investments, receivables, allowance_for_doubtful_accounts,
	withdrawals, sales, discounts, sales_returns_and_allowances []string
	db                   *sql.DB
	standard_days        = [7]string{"Saturday", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	adjusting_methods    = [4]string{"linear", "exponential", "logarithmic", "expire"}
	depreciation_methods = [3]string{"linear", "exponential", "logarithmic"}
	Now                  = time.Now()
)

func (s Financial_accounting) initialize() {
	db, _ = sql.Open(s.DriverName, s.DataSourceName)
	err := db.Ping()
	error_fatal(err)
	db.Exec("create database if not exists " + s.Database_name)
	_, err = db.Exec("USE " + s.Database_name)
	error_fatal(err)
	db.Exec("create table if not exists journal (date text,entry_number integer,account text,value real,price real,quantity real,barcode text,entry_expair text,description text,name text,employee_name text,entry_date text,reverse bool)")
	db.Exec("create table if not exists inventory (date text,account text,price real,quantity real,barcode text,entry_expair text,name text,employee_name text,entry_date text)")

	for _, i := range s.accounts {
		if !s.is_account_equal("", i.name) {
			log.Panic(i.name, " account does not ends in ''")
		}
		switch i.status {
		case "cash_and_cash_equivalent":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			cash_and_cash_equivalent = append(cash_and_cash_equivalent, i.name)
		case "fifo":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			fifo = append(fifo, i.name)
		case "lifo":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			lifo = append(lifo, i.name)
		case "wma":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			wma = append(wma, i.name)
		case "short_term_investments":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			short_term_investments = append(short_term_investments, i.name)
		case "receivables":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			receivables = append(receivables, i.name)
		case "allowance_for_doubtful_accounts":
			if !s.is_account_equal(s.assets, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			allowance_for_doubtful_accounts = append(allowance_for_doubtful_accounts, i.name)
		case "withdrawals":
			if !s.is_account_equal(s.retained_earnings, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			withdrawals = append(withdrawals, i.name)
		case "sales":
			if !s.is_account_equal(s.income_statement, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			sales = append(sales, i.name)
		case "discounts":
			if !s.is_account_equal(s.income_statement, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			discounts = append(discounts, i.name)
		case "sales_returns_and_allowances":
			if !s.is_account_equal(s.income_statement, i.name) {
				log.Panic(i.status, " for name ", i.name, " you can't use it with father name ", i.father)
			}
			sales_returns_and_allowances = append(sales_returns_and_allowances, i.name)
		case "debit":
			if s.is_account_equal(s.income_statement, i.name) {
				expenses = append(expenses, i.name)
			} else {
				debit_accounts = append(debit_accounts, i.name)
			}
		case "credit":
			if s.is_account_equal(s.income_statement, i.name) {
				revenues = append(revenues, i.name)
			} else {
				credit_accounts = append(credit_accounts, i.name)
			}
		default:
			log.Panic(i.status, " is not in [cash_and_cash_equivalent,fifo,lifo,wma,short_term_investments,receivables,allowance_for_doubtful_accounts,withdrawals,sales,discounts,sales_returns_and_allowances,debit,credit]")
		}
	}

	inventory = concat(fifo, lifo, wma).([]string)
	current_assets = concat(current_assets, inventory, cash_and_cash_equivalent, short_term_investments, receivables).([]string)
	assets_normal = concat(assets_normal, current_assets).([]string)
	assets_contra = concat(assets_contra, allowance_for_doubtful_accounts).([]string)
	liabilities_normal = concat(liabilities_normal).([]string)
	equity_normal = append(equity_normal)
	revenues = concat(revenues, sales).([]string)
	expenses = concat(expenses, sales_returns_and_allowances, discounts).([]string)
	temporary_debit_accounts = concat(withdrawals, expenses).([]string)
	temporary_accounts = concat(temporary_debit_accounts, revenues).([]string)
	debit_accounts = concat(debit_accounts, assets_normal, temporary_debit_accounts).([]string)
	credit_accounts = concat(credit_accounts, assets_contra, liabilities_normal, equity_normal, revenues).([]string)
	all_accounts := concat(debit_accounts, credit_accounts).([]string)
	for _, a := range []string{s.invoice_discount, s.retained_earnings, s.income_statement, s.assets} {
		if !is_in(a, all_accounts) {
			log.Panic(a, " is not on parameters accounts name")
		}
	}
	check_if_duplicates(all_accounts)
	check_accounts("account", "inventory", " is not on fifo lifo wma parameters accounts name", inventory)

	// entry_number := entry_number()
	// var array_to_insert []journal_tag
	// expair_expenses := journal_tag{Now.String(), entry_number, s.expair_expenses, 0, 0, 0, "", time.Time{}.String(), "to record the expiry of the goods automatically", "", "", Now.String(), false}
	// expair_goods, _ := db.Query("select account,price*quantity*-1,price,quantity*-1,barcode from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", Now.String())
	// for expair_goods.Next() {
	// 	tag := expair_expenses
	// 	expair_goods.Scan(&tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode)
	// 	expair_expenses.value -= tag.value
	// 	expair_expenses.quantity -= tag.quantity
	// 	array_to_insert = append(array_to_insert, tag)
	// }
	// expair_expenses.price = expair_expenses.value / expair_expenses.quantity
	// array_to_insert = append(array_to_insert, expair_expenses)
	// s.insert_to_database(array_to_insert, true, false, false)
	// db.Exec("delete from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", Now.String())
	db.Exec("delete from inventory where quantity=0")

	var journal [][]Account_value_quantity_barcode
	var double_entry []Account_value_quantity_barcode
	previous_entry_number := 1
	rows, _ := db.Query("select entry_number,account,value from journal")
	for rows.Next() {
		var entry_number int
		var tag Account_value_quantity_barcode
		rows.Scan(&entry_number, &tag.Account, &tag.value)
		if previous_entry_number != entry_number {
			check_debit_equal_credit(double_entry)
			journal = append(journal, double_entry)
			double_entry = []Account_value_quantity_barcode{}
		}
		double_entry = append(double_entry, tag)
		previous_entry_number = entry_number
	}
}

func (s Financial_accounting) journal_entry(array_of_entry []Account_value_quantity_barcode, insert, auto_completion bool, date time.Time, entry_expair time.Time, adjusting_method string,
	description string, name string, employee_name string, array_day_start_end []day_start_end) []journal_tag {

	if entry_expair.IsZero() == is_in(adjusting_method, adjusting_methods[:]) {
		log.Panic("check entry_expair => ", entry_expair, " and adjusting_method => ", adjusting_method, " should be in ", adjusting_methods)
	}

	if !entry_expair.IsZero() {
		check_dates(date, entry_expair)
	}

	array_of_entry = group_by_account_and_barcode(array_of_entry)
	array_of_entry = remove_zero_values(array_of_entry)
	for index, entry := range array_of_entry {
		if entry.Account == "" && entry.barcode == "" {
			log.Panic("can't find the account name if the barcode is empty in ", entry)
		}
		var tag string
		if entry.Account == "" {
			err := db.QueryRow("select account from journal where barcode=? limit 1", entry.barcode).Scan(&tag)
			if err != nil {
				log.Panic("the barcode is wrong for ", entry)
			}
			array_of_entry[index].Account = tag
		}
		if is_in(entry.Account, inventory) && !is_in(adjusting_method, []string{"expire", ""}) {
			log.Panic(entry.Account + " is in inventory you just can use expire or make it empty")
		}
	}

	for index, entry := range array_of_entry {
		costs := s.cost_flow(entry.Account, entry.quantity, entry.barcode, false)
		if costs != 0 {
			array_of_entry[index] = Account_value_quantity_barcode{entry.Account, -costs, entry.quantity, entry.barcode}
		}
		if auto_completion {
			for _, complement := range s.auto_complete_entries {
				if complement[0].account == entry.Account && (entry.quantity >= 0) == (complement[0].value_or_percent >= 0) {
					if costs == 0 {
						array_of_entry[index] = Account_value_quantity_barcode{complement[0].account, complement[0].price * entry.quantity, entry.quantity, ""}
					}
					for _, i := range complement[1:] {
						switch i.method {
						case "copy_abs":
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, math.Abs(array_of_entry[index].value), math.Abs(array_of_entry[index].quantity), ""})
						case "copy":
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, array_of_entry[index].value, array_of_entry[index].quantity, ""})
						case "quantity_ratio":
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, math.Abs(array_of_entry[index].quantity) * i.price * i.value_or_percent, math.Abs(array_of_entry[index].quantity) * i.value_or_percent, ""})
						case "value":
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, i.value_or_percent, i.value_or_percent / i.price, ""})
						default:
							log.Panic(i.method, "in the method field for ", i, " dose not exist you just can use copy_abs or copy or quantity_ratio or value")
						}
					}
				}
			}
		}
	}
	if auto_completion {
		var total_invoice_before_invoice_discount, discount float64
		for _, entry := range array_of_entry {
			if is_in(entry.Account, revenues) {
				total_invoice_before_invoice_discount += entry.value
			} else if is_in(entry.Account, discounts) {
				total_invoice_before_invoice_discount -= entry.value
			}
		}
		for _, i := range s.Invoice_discounts_list {
			if total_invoice_before_invoice_discount >= i[0] {
				discount = i[1]
			}
		}
		invoice_discount := discount_tax_calculator(total_invoice_before_invoice_discount, discount)
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{s.invoice_discount, invoice_discount, 1, ""})
	}

	array_of_entry = group_by_account_and_barcode(array_of_entry)
	array_of_entry = remove_zero_values(array_of_entry)
	var price_slice []float64
	for index, entry := range array_of_entry {
		if !is_in(entry.Account, equity_normal) {
			var account_balance float64
			db.QueryRow("select sum(value) from journal where account=? and date<?", entry.Account, Now.String()).Scan(&account_balance)
			if account_balance+entry.value < 0 {
				log.Panic("you cant enter ", entry, " because you have ", account_balance, " and that will make the balance of ", entry.Account, " negative ", account_balance+entry.value, " and that you just can do it in equity_normal accounts not other accounts")
			}
		}
		price_slice = append(price_slice, entry.value/entry.quantity)
		if price_slice[index] < 0 {
			log.Panic("the ", entry.value, " and ", entry.quantity, " for ", entry, " should be positive both or negative both")
		}
	}

	check_debit_equal_credit(array_of_entry)

	entry_number := entry_number()
	var array_to_insert []journal_tag
	for index, entry := range array_of_entry {
		array_to_insert = append(array_to_insert, journal_tag{
			date:          date.String(),
			entry_number:  entry_number,
			account:       entry.Account,
			value:         entry.value,
			price:         price_slice[index],
			quantity:      entry.quantity,
			barcode:       entry.barcode,
			entry_expair:  entry_expair.String(),
			description:   description,
			name:          name,
			employee_name: employee_name,
			entry_date:    Now.String(),
			reverse:       false,
		})
	}

	if is_in(adjusting_method, depreciation_methods[:]) {
		if len(array_day_start_end) == 0 {
			array_day_start_end = []day_start_end{
				{"saturday", 0, 0, 23, 59},
				{"sunday", 0, 0, 23, 59},
				{"monday", 0, 0, 23, 59},
				{"tuesday", 0, 0, 23, 59},
				{"wednesday", 0, 0, 23, 59},
				{"thursday", 0, 0, 23, 59},
				{"friday", 0, 0, 23, 59}}
		}
		for index, element := range array_day_start_end {
			array_day_start_end[index].day = strings.Title(element.day)
			switch {
			case !is_in(array_day_start_end[index].day, standard_days[:]):
				log.Panic("error ", element.day, " for ", element, " is not in ", standard_days)
			case element.start_hour < 0:
				log.Panic("error ", element.start_hour, " for ", element, " is < 0")
			case element.start_hour > 23:
				log.Panic("error ", element.start_hour, " for ", element, " is > 23")
			case element.start_minute < 0:
				log.Panic("error ", element.start_minute, " for ", element, " is < 0")
			case element.start_minute > 59:
				log.Panic("error ", element.start_minute, " for ", element, " is > 59")
			case element.end_hour < 0:
				log.Panic("error ", element.end_hour, " for ", element, " is < 0")
			case element.end_hour > 23:
				log.Panic("error ", element.end_hour, " for ", element, " is > 23")
			case element.end_minute < 0:
				log.Panic("error ", element.end_minute, " for ", element, " is < 0")
			case element.end_minute > 59:
				log.Panic("error ", element.end_minute, " for ", element, " is > 59")
			}
		}

		var day_start_end_date_minutes_array []day_start_end_date_minutes
		var total_minutes float64
		var previous_end_date, end time.Time
		delta_days := int(entry_expair.Sub(date).Hours()/24 + 1)
		year, month_sting, day := date.Date()
		for day_counter := 0; day_counter < delta_days; day_counter++ {
			for _, element := range array_day_start_end {
				if start := time.Date(year, month_sting, day+day_counter, element.start_hour, element.start_minute, 0, 0, time.Local); start.Weekday().String() == element.day {
					previous_end_date = end
					end = time.Date(year, month_sting, day+day_counter, element.end_hour, element.end_minute, 0, 0, time.Local)
					if start.After(end) {
						log.Panic("the start_hour and start_minute should be smaller than end_hour and end_minute for ", element)
					}
					if previous_end_date.After(start) {
						log.Panic("the end_hour and end_minute for ", element.day, " should be smaller than start_hour and start_minute for the second ", element)
					}
					minutes := end.Sub(start).Minutes()
					total_minutes += minutes
					day_start_end_date_minutes_array = append(day_start_end_date_minutes_array, day_start_end_date_minutes{element.day, start, end, minutes})
				}
			}
		}

		var adjusted_array_to_insert [][]journal_tag
		for _, entry := range array_to_insert {
			var value, value_counter, second_counter float64
			var one_account_adjusted_list []journal_tag
			total_value := math.Abs(entry.value)
			deprecation := math.Pow(total_value, 1/total_minutes)
			value_per_second := entry.value / total_minutes
			for index, element := range day_start_end_date_minutes_array {
				switch adjusting_method {
				case "linear":
					value = element.minutes * value_per_second
				case "exponential":
					value = math.Pow(deprecation, second_counter+element.minutes) - math.Pow(deprecation, second_counter)
				case "logarithmic":
					value = (total_value / math.Pow(deprecation, second_counter)) - (total_value / math.Pow(deprecation, second_counter+element.minutes))
				}
				second_counter += element.minutes

				quantity := value / entry.price
				if index >= delta_days-1 {
					value = math.Abs(total_value - value_counter)
					quantity = value / entry.price
				}
				value_counter += math.Abs(value)
				if entry.value < 0 {
					value = -math.Abs(value)
				}
				if entry.quantity < 0 {
					quantity = -math.Abs(quantity)
				}

				one_account_adjusted_list = append(one_account_adjusted_list, journal_tag{
					date:          element.start_date.String(),
					entry_number:  entry_number,
					account:       entry.account,
					value:         value,
					price:         entry.price,
					quantity:      quantity,
					barcode:       entry.barcode,
					entry_expair:  element.end_date.String(),
					description:   description,
					name:          name,
					employee_name: employee_name,
					entry_date:    Now.String(),
					reverse:       false,
				})
			}
			adjusted_array_to_insert = append(adjusted_array_to_insert, one_account_adjusted_list)
		}
		adjusted_array_to_insert = transpose(adjusted_array_to_insert)
		array_to_insert = []journal_tag{}
		for _, element := range adjusted_array_to_insert {
			array_to_insert = append(array_to_insert, element...)
		}
	}
	s.insert_to_database(array_to_insert, insert, insert, insert)
	return array_to_insert
}

func (s Financial_accounting) financial_statements(start_date, end_date time.Time, periods int) ([][]statement, []financial_analysis_statement, []map[string]map[string]map[string]map[string]float64) {
	check_dates(start_date, end_date)
	d1 := int(end_date.Sub(start_date).Hours() / 24)
	var journal []journal_tag
	rows, _ := db.Query("select date,entry_number,account,value,quantity from journal order by date")
	for rows.Next() {
		var entry journal_tag
		rows.Scan(&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.quantity)
		journal = append(journal, entry)
	}
	all_flows_for_all := []map[string]map[string]map[string]map[string]float64{}
	statments_maps := []map[string]*value_quantity{}
	var analysis []financial_analysis_statement
	for a := 0; a < periods; a++ {
		start_date_to_enter := start_date.AddDate(0, 0, -d1*a)
		end_date_to_enter := end_date.AddDate(0, 0, -d1*a)
		all_flows_for_all = append(all_flows_for_all, s.all_flows(journal, start_date_to_enter, end_date_to_enter))
		statments_maps = append(statments_maps, s.prepare_statment_map(journal, start_date_to_enter, end_date_to_enter))
		analysis = append(analysis, s.prepare_statment_analysis(journal, start_date_to_enter, end_date_to_enter))
	}
	var statements [][]statement
	for _, a := range statments_maps {
		statements = append(statements, s.prepare_statement(a, statments_maps[periods-1]))
	}
	return statements, analysis, all_flows_for_all
}

func (s Financial_accounting) invoice(array_of_journal_tag []journal_tag) []invoice_struct {
	m := map[string]*invoice_struct{}
	for _, entry := range array_of_journal_tag {
		var key string
		switch {
		case is_in(entry.account, assets_normal) && !is_in(entry.account, inventory) && entry.value > 0:
			key = "total"
		case is_in(entry.account, discounts):
			key = "total discounts"
		case is_in(entry.account, sales):
			key = entry.account
		default:
			continue
		}
		sums := m[key]
		if sums == nil {
			sums = &invoice_struct{}
			m[key] = sums
		}
		sums.value += entry.value
		sums.quantity += entry.quantity
	}
	invoice := []invoice_struct{}
	for k, v := range m {
		invoice = append(invoice, invoice_struct{k, v.value, v.value / v.quantity, v.quantity})
	}
	return invoice
}

func (s Financial_accounting) reverse_entry(number uint, start_date, end_date, entry_expair time.Time, employee_name string) {
	reverse_entry_number := entry_number()
	var array_of_entry_to_reverse []journal_tag
	array_of_journal_tag := select_journal(number, "", start_date, end_date)
	if len(array_of_journal_tag) == 0 {
		log.Panic("this entry not exist")
	}
	for _, entry := range array_of_journal_tag {
		if !entry.reverse {
			if s.parse_date(entry.date).Before(Now) {
				db.Exec("update journal set reverse=True where date=? and entry_number=? and account=? and value=? and price=? and quantity=? and barcode=? and entry_expair=? and description=? and name=? and employee_name=? and entry_date=? and reverse=?",
					entry.date, entry.entry_number, entry.account, entry.value, entry.price, entry.quantity, entry.barcode, entry.entry_expair, entry.description, entry.name, entry.employee_name, entry.entry_date, entry.reverse)
				entry.description = "(reverse entry for entry number " + strconv.Itoa(entry.entry_number) + " entered by " + entry.employee_name + " and revised by " + employee_name + ")"
				entry.date = Now.String()
				entry.entry_number = reverse_entry_number
				entry.value *= -1
				entry.quantity *= -1
				entry.entry_expair = entry_expair.String()
				entry.employee_name = employee_name
				entry.entry_date = Now.String()
				array_of_entry_to_reverse = append(array_of_entry_to_reverse, entry)
				weighted_average([]string{entry.account})
			} else {
				db.Exec("delete from journal where date=? and entry_number=? and account=? and value=? and price=? and quantity=? and barcode=? and entry_expair=? and description=? and name=? and employee_name=? and entry_date=? and reverse=?",
					entry.date, entry.entry_number, entry.account, entry.value, entry.price, entry.quantity, entry.barcode, entry.entry_expair, entry.description, entry.name, entry.employee_name, entry.entry_date, entry.reverse)
			}
		}
	}
	s.insert_to_database(array_of_entry_to_reverse, true, true, true)
}

func (s Financial_accounting) insert_to_database(array_of_journal_tag []journal_tag, insert_into_journal, insert_into_inventory, inventory_flow bool) {
	for _, entry := range array_of_journal_tag {
		if insert_into_journal {
			db.Exec("insert into journal(date,entry_number,account,value,price,quantity,barcode,entry_expair,description,name,employee_name,entry_date,reverse) values (?,?,?,?,?,?,?,?,?,?,?,?,?)",
				&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.price, &entry.quantity, &entry.barcode,
				&entry.entry_expair, &entry.description, &entry.name, &entry.employee_name, &entry.entry_date, &entry.reverse)
		}
		if is_in(entry.account, inventory) {
			costs := s.cost_flow(entry.account, entry.quantity, entry.barcode, inventory_flow)
			if insert_into_inventory && costs == 0 {
				db.Exec("insert into inventory(date,account,price,quantity,barcode,entry_expair,name,employee_name,entry_date)values (?,?,?,?,?,?,?,?,?)",
					&entry.date, &entry.account, &entry.price, &entry.quantity, &entry.barcode, &entry.entry_expair, &entry.name, &entry.employee_name, &entry.entry_date)

			}
		}
	}
}

func (s Financial_accounting) cost_flow(account string, quantity float64, barcode string, insert bool) float64 {
	var order_by_date_asc_or_desc string
	switch {
	case quantity > 0:
		return 0
	case is_in(account, fifo):
		order_by_date_asc_or_desc = "asc"
	case is_in(account, lifo):
		order_by_date_asc_or_desc = "desc"
	case is_in(account, wma):
		weighted_average([]string{account})
		order_by_date_asc_or_desc = "asc"
	default:
		return 0
	}
	rows, _ := db.Query("select price,quantity from inventory where quantity>0 and account=? and barcode=? order by date "+order_by_date_asc_or_desc, account, barcode)
	var inventory []journal_tag
	for rows.Next() {
		var tag journal_tag
		rows.Scan(&tag.price, &tag.quantity)
		inventory = append(inventory, tag)
	}
	quantity = math.Abs(quantity)
	quantity_count := quantity
	var costs float64
	for _, item := range inventory {
		if item.quantity > quantity_count {
			costs += item.price * quantity_count
			if insert {
				db.Exec("update inventory set quantity=quantity-? where account=? and price=? and quantity=? and barcode=? order by date "+order_by_date_asc_or_desc+" limit 1", quantity_count, account, item.price, item.quantity, barcode)
			}
			quantity_count = 0
			break
		}
		if item.quantity <= quantity_count {
			costs += item.price * item.quantity
			if insert {
				db.Exec("delete from inventory where account=? and price=? and quantity=? and barcode=? order by date "+order_by_date_asc_or_desc+" limit 1", account, item.price, item.quantity, barcode)
			}
			quantity_count -= item.quantity
		}
	}
	if quantity_count != 0 {
		log.Panic("you order ", quantity, " but you have ", quantity-quantity_count, " ", account, " with barcode ", barcode)
	}
	return costs
}

func (s Financial_accounting) all_flows(journal []journal_tag, start_date, end_date time.Time) map[string]map[string]map[string]map[string]float64 {
	var one_compound_entry []journal_tag
	var previous_date string
	all_flows := map[string]map[string]map[string]map[string]float64{}
	for _, entry := range journal {
		date := s.parse_date(entry.date)
		if previous_date != entry.date {
			sum_flows(one_compound_entry, all_flows)
			one_compound_entry = []journal_tag{}
		}
		if date.After(start_date) && date.Before(end_date) {
			one_compound_entry = append(one_compound_entry, entry)
		}
		previous_date = entry.date
	}
	sum_flows(one_compound_entry, all_flows)
	return all_flows
}

func sum_flows(one_compound_entry []journal_tag, all_flows map[string]map[string]map[string]map[string]float64) {
	for _, a := range one_compound_entry {
		if all_flows[a.account] == nil {
			all_flows[a.account] = map[string]map[string]map[string]float64{}
		}
		for _, b := range one_compound_entry {
			if all_flows[a.account][b.account] == nil {
				all_flows[a.account][b.account] = map[string]map[string]float64{}
			}
			if all_flows[a.account][b.account][b.name] == nil {
				all_flows[a.account][b.account][b.name] = map[string]float64{}
			}
			if all_flows[a.account][b.account][b.name] == nil {
				all_flows[a.account][b.account][b.name] = map[string]float64{}
			}
			all_flows[a.account][b.account][b.name]["value_increase"] += b.value
			all_flows[a.account][b.account][b.name]["quantity_increase"] += b.quantity
			if b.account == a.account {
				if b.value < 0 {
					all_flows[a.account][b.account][b.name]["outflow"] -= b.value
				} else {
					all_flows[a.account][b.account][b.name]["inflow"] += b.value
				}
				all_flows[a.account][b.account][b.name]["flow"] += b.value
			} else if is_in(b.account, debit_accounts) == is_in(a.account, debit_accounts) {
				if b.value >= 0 {
					all_flows[a.account][b.account][b.name]["outflow"] += b.value
				} else {
					all_flows[a.account][b.account][b.name]["inflow"] -= b.value
				}
				all_flows[a.account][b.account][b.name]["flow"] -= b.value
			} else {
				if b.value < 0 {
					all_flows[a.account][b.account][b.name]["outflow"] -= b.value
				} else {
					all_flows[a.account][b.account][b.name]["inflow"] += b.value
				}
				all_flows[a.account][b.account][b.name]["flow"] += b.value
			}
		}
	}
}

func (s Financial_accounting) prepare_statment_map(journal []journal_tag, start_date, end_date time.Time) map[string]*value_quantity {
	var cash []journal_tag
	var ok bool
	var previous_date string
	journal_map := map[string]*value_quantity{s.retained_earnings: {0, 0, 0, 0}}
	for _, entry := range journal {
		key_journal := entry.account
		sum_journal := journal_map[key_journal]
		if sum_journal == nil {
			sum_journal = &value_quantity{}
			journal_map[key_journal] = sum_journal
		}
		date := s.parse_date(entry.date)
		if previous_date != entry.date {
			sum_cash_flow(ok, cash, journal_map)
			cash = []journal_tag{}
			ok = false
		}
		previous_date = entry.date
		if date.Before(start_date) {
			switch {
			case is_in(entry.account, revenues):
				sum_journal := journal_map[s.retained_earnings]
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
			case is_in(entry.account, temporary_debit_accounts):
				sum_journal := journal_map[s.retained_earnings]
				sum_journal.value -= entry.value
				sum_journal.quantity -= entry.quantity
			default:
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
			}
		}
		if date.After(start_date) && date.Before(end_date) {
			if is_in(entry.account, cash_and_cash_equivalent) {
				ok = true
			} else {
				cash = append(cash, entry)
			}
			sum_journal.value += entry.value
			sum_journal.quantity += entry.quantity
		}
	}
	sum_cash_flow(ok, cash, journal_map)
	new_statement_map := map[string]*value_quantity{}
	for key, v := range journal_map {
		var last_name string
		key1 := key
		for {
			for _, a := range s.accounts {
				if a.name == key {
					key = a.father
					sum_statement := new_statement_map[a.name]
					if sum_statement == nil {
						sum_statement = &value_quantity{}
						new_statement_map[a.name] = sum_statement
					}
					if is_in(key1, debit_accounts) == is_in(a.name, debit_accounts) {
						sum_statement.value += v.value
						sum_statement.quantity += v.quantity
					} else {
						sum_statement.value -= v.value
						sum_statement.quantity -= v.quantity
					}
					sum_statement.inflow += v.inflow
					sum_statement.outflow += v.outflow
				}
			}
			if last_name == key {
				break
			}
			last_name = key
		}
	}
	return new_statement_map
}

func (s Financial_accounting) prepare_statment_analysis(journal []journal_tag, start_date, end_date time.Time) financial_analysis_statement {
	var v_current_assets, v_current_liabilities, v_cash, v_short_term_investments, v_net_receivables, v_net_credit_sales,
		v_average_net_receivables, v_cost_of_goods_sold, v_average_inventory, v_net_income, v_net_sales, v_average_assets,
		v_preferred_dividends, v_average_common_stockholders_equity, v_weighted_average_common_shares_outstanding,
		v_market_price_per_shares_outstanding, v_cash_dividends, v_total_debt, v_total_assets,
		v_ebitda, v_interest_expense float64

	// for _, entry := range journal {
	// 	date := s.parse_date(entry.date)
	// 	if date.After(end_date) {
	// 		continue
	// 	}
	// 	var index int
	// 	var daddies []string
	// 	var last_name, status string
	// 	name := entry.account
	// 	for {
	// 		for _, a := range s.accounts {
	// 			if a.name == name {
	// 				name = a.father
	// 				daddies = append(daddies, name)
	// 				if index == 0 {
	// 					status = a.status
	// 				}
	// 				index++
	// 			}
	// 		}
	// 		if last_name == name {
	// 			break
	// 		}
	// 		last_name = name
	// 	}
	// 	fmt.Println(entry.account, daddies, status)
	// 	// this need period
	// 	if date.After(start_date) {
	// 		// this need average
	// 		if true { //"average_net_receivables"
	// 			v_average_net_receivables += entry.value / 2
	// 		}
	// 		if true { //"average_inventory"
	// 			v_average_inventory += entry.value / 2
	// 		}
	// 		if true { //"average_assets"
	// 			v_average_assets += entry.value / 2
	// 		}
	// 		if true { //"average_common_stockholders_equity"
	// 			v_average_common_stockholders_equity += entry.value / 2
	// 		}
	// 		if true { //"net_income"
	// 			v_net_income += entry.value
	// 		}
	// 		if true { //"net_sales"
	// 			v_net_sales += entry.value
	// 		}
	// 		// this need flow
	// 		if is_in(status, []string{"fifo", "lifo", "wma"}) && entry.value < 0 { //"cost_of_goods_sold"
	// 			v_cost_of_goods_sold -= entry.value
	// 		}
	// 		if status == "sales" { //"net_credit_sales"
	// 			v_net_credit_sales += entry.value
	// 		}
	// 		if is_in(status, []string{"cash_and_cash_equivalent"}) && entry.value < 0 { //"cash_dividends"
	// 			v_cash_dividends -= entry.value
	// 		}
	// 	}
	// 	// this need sum
	// 	// this need average
	// 	if true { //"average_net_receivables"
	// 		v_average_net_receivables += entry.value
	// 	}
	// 	if true { //"average_inventory"
	// 		v_average_inventory += entry.value
	// 	}
	// 	if true { //"average_assets"
	// 		v_average_assets += entry.value
	// 	}
	// 	if true { //"average_common_stockholders_equity"
	// 		v_average_common_stockholders_equity += entry.value
	// 	}
	// 	if true { //is_in(entry.account, current_assets)
	// 		v_current_assets += entry.value
	// 	}
	// 	if true { //"current_liabilities"
	// 		v_current_liabilities += entry.value
	// 	}
	// 	if true { //"cash"
	// 		v_cash += entry.value
	// 	}
	// 	if true { //"short_term_investments"
	// 		v_short_term_investments += entry.value
	// 	}
	// 	if true { //"net_receivables"
	// 		v_net_receivables += entry.value
	// 	}
	// 	if true { //"preferred_dividends"
	// 		v_preferred_dividends += entry.value
	// 	}
	// 	if true { //"weighted_average_common_shares_outstanding"
	// 		v_weighted_average_common_shares_outstanding += entry.value
	// 	}
	// 	if true { //"market_price_per_shares_outstanding"
	// 		v_market_price_per_shares_outstanding += entry.value
	// 	}
	// 	if true { //"total_debt"
	// 		v_total_debt += entry.value
	// 	}
	// 	if true { //"total_assets"
	// 		v_total_assets += entry.value
	// 	}
	// 	if true { //"income_before_income_taxes_and_interest_expense"
	// 		v_ebitda += entry.value
	// 	}
	// 	if true { //"interest_expense"
	// 		v_interest_expense += entry.value
	// 	}
	// }
	return financial_analysis{
		current_assets:                      v_current_assets,
		current_liabilities:                 v_current_liabilities,
		cash:                                v_cash,
		short_term_investments:              v_short_term_investments,
		net_receivables:                     v_net_receivables,
		net_credit_sales:                    v_net_credit_sales,
		average_net_receivables:             v_average_net_receivables,
		cost_of_goods_sold:                  v_cost_of_goods_sold,
		average_inventory:                   v_average_inventory,
		net_income:                          v_net_income,
		net_sales:                           v_net_sales,
		average_assets:                      v_average_assets,
		preferred_dividends:                 v_preferred_dividends,
		average_common_stockholders_equity:  v_average_common_stockholders_equity,
		market_price_per_shares_outstanding: v_market_price_per_shares_outstanding,
		cash_dividends:                      v_cash_dividends,
		total_debt:                          v_total_debt,
		total_assets:                        v_total_assets,
		ebitda:                              v_ebitda,
		interest_expense:                    v_interest_expense,
		weighted_average_common_shares_outstanding: v_weighted_average_common_shares_outstanding,
	}.financial_analysis_statement()
}

func (s Financial_accounting) prepare_statement(statement_map, statement_map_base map[string]*value_quantity) []statement {
	var statement_sheet []statement
	var total_assets, total_sales, total float64
	for key, v := range statement_map {
		if key == s.assets {
			total_assets = v.value
		}
		if is_in(key, sales) {
			total_sales += v.value
		}
	}
	for key, v := range statement_map {
		if !s.is_account_equal(s.income_statement, key) {
			total = total_assets
		} else {
			total = total_sales
		}
		var base_year_amount float64
		for key1, v1 := range statement_map_base {
			if key == key1 {
				base_year_amount = v1.value
				break
			}
		}
		statement_sheet = append(statement_sheet, statement{
			account:                            key,
			value:                              v.value,
			price:                              v.value / v.quantity,
			quantity:                           v.quantity,
			percent:                            v.value / total,
			inflow:                             v.inflow,
			outflow:                            v.outflow,
			flow:                               v.inflow - v.outflow,
			flow_percent:                       0,
			change_in_amount_since_base_period: v.value - base_year_amount,
			current_results_in_relation_to_base_period: v.value / base_year_amount,
		})
	}
	var index int
	for index < len(statement_sheet) {
		if (statement_sheet[index].value == 0 || math.IsNaN(statement_sheet[index].value)) &&
			(statement_sheet[index].price == 0 || math.IsNaN(statement_sheet[index].price)) &&
			(statement_sheet[index].quantity == 0 || math.IsNaN(statement_sheet[index].quantity)) &&
			(statement_sheet[index].percent == 0 || math.IsNaN(statement_sheet[index].percent)) &&
			(statement_sheet[index].inflow == 0 || math.IsNaN(statement_sheet[index].inflow)) &&
			(statement_sheet[index].outflow == 0 || math.IsNaN(statement_sheet[index].outflow)) &&
			(statement_sheet[index].flow == 0 || math.IsNaN(statement_sheet[index].flow)) &&
			(statement_sheet[index].flow_percent == 0 || math.IsNaN(statement_sheet[index].flow_percent)) &&
			(statement_sheet[index].change_in_amount_since_base_period == 0 || math.IsNaN(statement_sheet[index].change_in_amount_since_base_period)) &&
			(statement_sheet[index].current_results_in_relation_to_base_period == 0 || math.IsNaN(statement_sheet[index].current_results_in_relation_to_base_period)) {
			statement_sheet = append(statement_sheet[:index], statement_sheet[index+1:]...)
		} else {
			index++
		}
	}
	var indexa int
	for _, a := range s.accounts {
		for indexb, b := range statement_sheet {
			if a.name == b.account {
				statement_sheet[indexa], statement_sheet[indexb] = statement_sheet[indexb], statement_sheet[indexa]
				indexa++
				break
			}
		}
	}
	return statement_sheet
}

func (s Financial_accounting) parse_date(string_date string) time.Time {
	for _, i := range s.date_layout {
		date, err := time.Parse(i, string_date)
		if err == nil {
			return date
		}
	}
	return time.Time{}
}

func (s Financial_accounting) is_account_equal(father, name string) bool {
	var last_name string
	for {
		for _, a := range s.accounts {
			if a.name == name {
				name = a.father
			}
			if father == name {
				return true
			}
		}
		if last_name == name {
			break
		}
		last_name = name
	}
	return false
}

func (s Financial_accounting) return_status(name string) string {
	for _, a := range s.accounts {
		if a.name == name {
			return a.status
		}
	}
	return ""
}

func sum_cash_flow(ok bool, cash []journal_tag, journal_map map[string]*value_quantity) {
	if ok {
		for _, entry := range cash {
			sum_journal := journal_map[entry.account]
			if is_in(entry.account, debit_accounts) {
				if entry.value >= 0 {
					sum_journal.outflow += entry.value
				} else {
					sum_journal.inflow -= entry.value
				}
			} else {
				if entry.value < 0 {
					sum_journal.outflow -= entry.value
				} else {
					sum_journal.inflow += entry.value
				}
			}
		}
	}
}

func select_journal(entry_number uint, account string, start_date, end_date time.Time) []journal_tag {
	var journal []journal_tag
	var rows *sql.Rows
	switch {
	case entry_number != 0 && account == "":
		rows, _ = db.Query("select * from journal where date>? and date<? and entry_number=? order by date", start_date.String(), end_date.String(), entry_number)
	case entry_number == 0 && account != "":
		rows, _ = db.Query("select * from journal where date>? and date<? and account=? order by date", start_date.String(), end_date.String(), account)
	default:
		log.Panic("should be one of these entry_number != 0 && account == '' or entry_number == 0 && account != '' ")
	}
	for rows.Next() {
		var tag journal_tag
		rows.Scan(&tag.date, &tag.entry_number, &tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode, &tag.entry_expair, &tag.description, &tag.name, &tag.employee_name, &tag.entry_date, &tag.reverse)
		journal = append(journal, tag)
	}
	return journal
}

func weighted_average(array_of_accounts []string) {
	for _, account := range array_of_accounts {
		db.Exec("update inventory set price=(select sum(value)/sum(quantity) from journal where account=?) where account=?", account, account)
	}
}

func entry_number() int {
	var tag int
	err := db.QueryRow("select max(entry_number) from journal").Scan(&tag)
	if err != nil {
		tag = 0
	}
	return tag + 1
}

func group_by_account_and_barcode(array_of_entry []Account_value_quantity_barcode) []Account_value_quantity_barcode {
	type Account_barcode struct {
		Account, barcode string
	}
	g := map[Account_barcode]*Account_value_quantity_barcode{}
	for _, v := range array_of_entry {
		key := Account_barcode{v.Account, v.barcode}
		sums := g[key]
		if sums == nil {
			sums = &Account_value_quantity_barcode{}
			g[key] = sums
		}
		sums.value += v.value
		sums.quantity += v.quantity
	}
	array_of_entry = []Account_value_quantity_barcode{}
	for key, v := range g {
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{key.Account, v.value, v.quantity, key.barcode})
	}
	return array_of_entry
}

func remove_zero_values(array_of_entry []Account_value_quantity_barcode) []Account_value_quantity_barcode {
	var index int
	for index < len(array_of_entry) {
		if array_of_entry[index].value == 0 || array_of_entry[index].quantity == 0 {
			// fmt.Println(array_of_entry[index], " is removed because one of the values is 0")
			array_of_entry = append(array_of_entry[:index], array_of_entry[index+1:]...)
		} else {
			index++
		}
	}
	return array_of_entry
}

func change_account_name(name, new_name string) {
	var tag string
	err := db.QueryRow("select account from journal where account=? limit 1", new_name).Scan(&tag)
	if err == nil {
		log.Panic("you can't change the name of [", name, "] to [", new_name, "] as new name because it used")
	} else {
		db.Exec("update journal set account=? where account=?", new_name, name)
		db.Exec("update inventory set account=? where account=?", new_name, name)
	}
}

func is_in(element string, elements []string) bool {
	for _, a := range elements {
		if a == element {
			return true
		}
	}
	return false
}

func check_debit_equal_credit(array_of_entry []Account_value_quantity_barcode) {
	var zero float64
	var debit_number, credit_number int
	for _, entry := range array_of_entry {
		switch {
		case is_in(entry.Account, debit_accounts):
			zero += entry.value
			if entry.value >= 0 {
				debit_number++
			} else {
				credit_number++
			}
		case is_in(entry.Account, credit_accounts):
			zero -= entry.value
			if entry.value <= 0 {
				debit_number++
			} else {
				credit_number++
			}
		default:
			log.Panic(entry.Account, " is not on parameters accounts name")
		}
	}
	if (debit_number != 1) && (credit_number != 1) {
		// log.Panic("should be one credit or one debit in the entry")
	}
	if zero != 0 {
		log.Panic(zero, " not equal 0 if the number>0 it means debit overstated else credit overstated debit-credit should equal zero ", array_of_entry)
	}
}

func check_accounts(column, table, panic string, elements []string) {
	results, err := db.Query("select " + column + " from " + table)
	error_fatal(err)
	for results.Next() {
		var tag string
		results.Scan(&tag)
		if !is_in(tag, elements) {
			log.Panic(tag + panic)
		}
	}
}

func error_fatal(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func check_dates(start_date, end_date time.Time) {
	if start_date.After(end_date) {
		log.Panic("please enter the start_date<=end_date")
	}
}

func check_if_duplicates(slice_of_elements []string) {
	var set_of_elems, duplicated_element []string
	for _, element := range slice_of_elements {
		for _, b := range set_of_elems {
			if b == element {
				duplicated_element = append(duplicated_element, element)
				break
			}
		}
		set_of_elems = append(set_of_elems, element)
	}
	if len(duplicated_element) != 0 {
		log.Panic(duplicated_element, " is duplicated values in the fields of Financial_accounting and that make error. you should remove the duplicate")
	}
}

func concat(args ...interface{}) interface{} {
	n := 0
	for _, arg := range args {
		n += reflect.ValueOf(arg).Len()
	}
	v := reflect.MakeSlice(reflect.TypeOf(args[0]), 0, n)
	for _, arg := range args {
		v = reflect.AppendSlice(v, reflect.ValueOf(arg))
	}
	return v.Interface()
}

func discount_tax_calculator(price, discount_tax float64) float64 {
	if discount_tax < 0 {
		discount_tax = math.Abs(discount_tax)
	} else if discount_tax > 0 {
		discount_tax = price * discount_tax
	}
	return discount_tax
}

func transpose(slice [][]journal_tag) [][]journal_tag {
	xl := len(slice[0])
	yl := len(slice)
	result := make([][]journal_tag, xl)
	for i := range result {
		result[i] = make([]journal_tag, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			result[i][j] = slice[j][i]
		}
	}
	return result
}

func (s financial_analysis) financial_analysis_statement() financial_analysis_statement {
	current_ratio := s.current_assets / s.current_liabilities
	acid_test := (s.cash + s.short_term_investments + s.net_receivables) / s.current_liabilities
	receivables_turnover := s.net_credit_sales / s.average_net_receivables
	inventory_turnover := s.cost_of_goods_sold / s.average_inventory
	profit_margin := s.net_income / s.net_sales
	asset_turnover := s.net_sales / s.average_assets
	return_on_assets := s.net_income / s.average_assets
	return_on_common_stockholders_equity := (s.net_income - s.preferred_dividends) / s.average_common_stockholders_equity
	earnings_per_share := (s.net_income - s.preferred_dividends) / s.weighted_average_common_shares_outstanding
	price_earnings_ratio := s.market_price_per_shares_outstanding / earnings_per_share
	payout_ratio := s.cash_dividends / s.net_income
	debt_to_total_assets_ratio := s.total_debt / s.total_assets
	times_interest_earned := s.ebitda / s.interest_expense
	return financial_analysis_statement{
		current_ratio:                        current_ratio,
		acid_test:                            acid_test,
		receivables_turnover:                 receivables_turnover,
		inventory_turnover:                   inventory_turnover,
		profit_margin:                        profit_margin,
		asset_turnover:                       asset_turnover,
		return_on_assets:                     return_on_assets,
		return_on_common_stockholders_equity: return_on_common_stockholders_equity,
		earnings_per_share:                   earnings_per_share,
		price_earnings_ratio:                 price_earnings_ratio,
		payout_ratio:                         payout_ratio,
		debt_to_total_assets_ratio:           debt_to_total_assets_ratio,
		times_interest_earned:                times_interest_earned,
	}
}

func (s Managerial_Accounting) decrease() float64 {
	return s.beginning_balance + s.increase - s.ending_balance
}

func (s Managerial_Accounting) cost_of_goods_sold() float64 {
	return s.decrease() - s.decreases_in_account_caused_by_not_sell
}

func (s Managerial_Accounting) cost_volume_profit_slice() []cvp_statistics {
	var h []cvp_statistics
	length_fixed_cost := len(s.cvp[0].portions)
	total_portions := make([]float64, length_fixed_cost)
	for _, i := range s.cvp {
		if length_fixed_cost != len(i.portions) {
			log.Panic("length of portions and fixed_cost in overhead that have portions percent_method, should be all the same length")
		}
		for index, i := range i.portions {
			total_portions[index] += i
		}
	}
	for _, i := range s.overhead {
		if length_fixed_cost != len(i.fixed_cost) && i.percent_method == "portions" {
			log.Panic("length of portions and fixed_cost in overhead that have portions percent_method, should be all the same length")
		}
	}
	for _, i := range s.cvp {
		h = append(h, cost_volume_profit(i.name, i.units, i.selling_price_per_unit, i.variable_cost_per_unit, i.fixed_cost))
	}
	for _, a := range s.overhead {
		totals := total_cost_volume_profit(h)
		var total_overhead_cost float64
		for _, i := range a.fixed_cost {
			total_overhead_cost += i
		}
		for indexb, b := range h {
			var percent float64
			switch a.percent_method {
			case "units":
				percent = b.units / totals.units
			case "fixed_cost":
				percent = b.fixed_cost / totals.fixed_cost
			case "mixed_cost":
				percent = b.mixed_cost / totals.mixed_cost
			case "sales":
				percent = b.sales / totals.sales
			case "profit":
				percent = b.profit / totals.profit
			case "contribution_margin":
				percent = b.contribution_margin / totals.contribution_margin
			case "1":
				percent = 1
			case "portions":
				var sum_portions_cost float64
				for indexc, c := range s.cvp[indexb].portions {
					sum_portions_cost += c / total_portions[indexc] * a.fixed_cost[indexc]
				}
				percent = sum_portions_cost / total_overhead_cost
			default:
				log.Panic(a.percent_method, " is not in [units,fixed_cost,mixed_cost,sales,profit,contribution_margin,1,portions]")
			}
			if math.IsNaN(percent) {
				percent = 0
			}
			h[indexb] = cost_volume_profit(b.name, b.units, b.selling_price_per_unit, b.variable_cost_per_unit, b.fixed_cost+(percent*total_overhead_cost))
		}
	}
	return append(h, total_cost_volume_profit(h))
}

func (s Managerial_Accounting) equivalent_units() float64 {
	return s.number_of_partially_completed_units * s.percentage_completion
}

func (s Managerial_Accounting) equivalent_units_of_production_weighted_average_method() float64 {
	return s.units_transferred_to_the_next_department_or_to_finished_goods + s.equivalent_units_in_ending_work_in_process_inventory
}

func (s Managerial_Accounting) cost_per_equivalent_unit_weighted_average_method() float64 {
	return (s.cost_of_beginning_work_in_process_inventory + s.cost_added_during_the_period) / s.equivalent_units_of_production_weighted_average_method()
}

func (s Managerial_Accounting) equivalent_units_of_production_fifo_method() float64 {
	return s.equivalent_units_of_production_weighted_average_method() - s.equivalent_units_in_beginning_work_in_process_inventory
}

func (s Managerial_Accounting) equivalent_units_to_complete_beginning_work_in_process_inventory() float64 {
	return s.equivalent_units_in_beginning_work_in_process_inventory * (1 - s.percentage_completion)
}

func (s Managerial_Accounting) cost_per_equivalent_unit_fifo_method() float64 {
	return s.cost_added_during_the_period / s.equivalent_units_of_production_fifo_method()
}

// return variable_cost
func (s Managerial_Accounting) high_low() float64 {
	var y2, y1, x2, x1 float64
	for _, i := range s.points_activity_level_and_cost_at_the_activity_level {
		if i[0] >= x2 {
			x2 = i[0]
			y2 = i[1]
		} else if i[0] < x1 {
			x1 = i[0]
			y1 = i[1]
		}
	}
	return (y2 - y1) / (x2 - x1)
}

// return variable_cost and fixed_cost
func (s Managerial_Accounting) least_squares_regression() (float64, float64) {
	var sum_x, sum_y, sum_x_quadratic, sum_xy float64
	for _, i := range s.points_activity_level_and_cost_at_the_activity_level {
		sum_x += i[0]
		sum_y += i[1]
		sum_x_quadratic += math.Pow(i[0], 2)
		sum_xy += i[0] * i[1]
	}
	n := float64(len(s.points_activity_level_and_cost_at_the_activity_level))
	m := (n*sum_xy - sum_x*sum_y) / ((n * sum_x_quadratic) - math.Pow(sum_x, 2))
	b := (sum_y - (m * sum_x)) / n
	return m, b
}

func cost_volume_profit(name string, units, selling_price_per_unit, variable_cost_per_unit, fixed_cost float64) cvp_statistics {
	mixed_cost := fixed_cost + variable_cost_per_unit*units
	mixed_cost_per_unit := mixed_cost / units
	sales := selling_price_per_unit * units
	profit := selling_price_per_unit*units - mixed_cost
	profit_per_unit := profit / units
	contribution_margin_per_unit := selling_price_per_unit - variable_cost_per_unit
	contribution_margin := contribution_margin_per_unit * units
	contribution_margin_ratio := contribution_margin_per_unit / selling_price_per_unit
	break_even_in_unit := fixed_cost / contribution_margin_per_unit
	break_even_in_sales := break_even_in_unit * selling_price_per_unit
	degree_of_operating_leverage := contribution_margin / profit
	return cvp_statistics{
		name:                         name,
		units:                        units,
		selling_price_per_unit:       selling_price_per_unit,
		variable_cost_per_unit:       variable_cost_per_unit,
		fixed_cost:                   fixed_cost,
		mixed_cost:                   mixed_cost,
		mixed_cost_per_unit:          mixed_cost_per_unit,
		sales:                        sales,
		profit:                       profit,
		profit_per_unit:              profit_per_unit,
		contribution_margin_per_unit: contribution_margin_per_unit,
		contribution_margin:          contribution_margin,
		contribution_margin_ratio:    contribution_margin_ratio,
		break_even_in_unit:           break_even_in_unit,
		break_even_in_sales:          break_even_in_sales,
		degree_of_operating_leverage: degree_of_operating_leverage,
	}
}

func total_cost_volume_profit(cvp_statistics_slice []cvp_statistics) cvp_statistics {
	var units, selling_price_per_unit, variable_cost_per_unit, fixed_cost float64
	for _, a := range cvp_statistics_slice {
		units += a.units
		selling_price_per_unit += a.selling_price_per_unit * a.units
		variable_cost_per_unit += a.variable_cost_per_unit * a.units
		fixed_cost += a.fixed_cost
	}
	return cost_volume_profit("total", units, selling_price_per_unit/units, variable_cost_per_unit/units, fixed_cost)
}

func target_sales(target_profit, fixed_cost, contribution_margin_ratio float64) float64 {
	return (target_profit + fixed_cost) / contribution_margin_ratio
}

func main() {
	v := Financial_accounting{
		date_layout:       []string{"2006-01-02 15:04:05.999999999 -0700 +03 m=+0.999999999", "2006-01-02 15:04:05.999999999 -0700 +03"},
		DriverName:        "mysql",
		DataSourceName:    "hashem:hashem@tcp(localhost)/",
		Database_name:     "acc",
		invoice_discount:  "invoice_discount",
		retained_earnings: "retained_earnings",
		income_statement:  "income_statement",
		assets:            "assets",
		accounts: []account{
			{"debit", "", "assets"},
			{"cash_and_cash_equivalent", "assets", "cash"},
			{"debit", "assets", "inventory"},
			{"fifo", "inventory", "book"},
			{"credit", "", "Liabilities"},
			{"credit", "Liabilities", "tax"},
			{"credit", "", "equity"},
			{"credit", "equity", "retained_earnings"},
			{"credit", "equity", "income_statement"},
			{"credit", "income_statement", "Revenues"},
			{"sales", "Revenues", "service revenue"},
			{"sales", "Revenues", "revenue of book"},
			{"debit", "income_statement", "Expenses"},
			{"debit", "Expenses", "expair_expenses"},
			{"debit", "Expenses", "discount of book"},
			{"debit", "Expenses", "invoice_discount"},
			{"debit", "Expenses", "service_discount"},
			{"debit", "Expenses", "cost of book"},
			{"debit", "Expenses", "tax of book"},
			{"debit", "Expenses", "tax of service revenue"},
			{"debit", "Expenses", "invoice_tax"}},
		Invoice_discounts_list: [][2]float64{{5, -10}},
		auto_complete_entries: [][]account_method_value_price{
			{{"service revenue", "quantity_ratio", 0, 10}, {"tax of service revenue", "value", 1, 1}, {"tax", "value", 1, 1}, {"service_discount", "value", 1, 1}},
			{{"book", "quantity_ratio", -1, 0}, {"revenue of book", "quantity_ratio", 1, 10}, {"cost of book", "copy_abs", 0, 0}, {"tax of book", "value", 1, 1}, {"tax", "value", 1, 1}, {"discount of book", "value", 1, 1}}},
	}
	v.initialize()
	v.journal_entry([]Account_value_quantity_barcode{{"service revenue", 10, 100, ""}, {"cash", 989, 989, ""}}, false, true, Now,
		time.Time{}, "", "", "yasa", "hashem", []day_start_end{})
	// entry := select_journal(0, "cash", time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local))
	// fmt.Println(v.invoice(entry))
	// reverse_entry(2, time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local), "hashem")
	// r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// for _, i := range entry {
	// 	fmt.Fprintln(r, "\t", i.date, "\t", i.entry_number, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.barcode, "\t", i.entry_expair, "\t", i.description, "\t", i.name, "\t", i.employee_name, "\t", i.entry_date, "\t", i.reverse)
	// }
	// r.Flush()
	balance_sheet, _, all_flows_for_all := v.financial_statements(
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local),
		time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local),
		1)
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	for index, a := range balance_sheet {
		for _, b := range a {
			fmt.Fprintln(w, index, "\t", b.account, "\t", b.value, "\t", b.price, "\t", b.quantity, "\t", b.percent, "\t", b.inflow, "\t", b.outflow, "\t", b.flow, "\t", b.change_in_amount_since_base_period, "\t", b.current_results_in_relation_to_base_period)
		}
	}
	w.Flush()
	// for _, a := range financial_analysis_statement {
	// 	fmt.Println(a)
	// }
	r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	for _, v := range all_flows_for_all {
		for keya, a := range v {
			for keyb, b := range a {
				for keyc, c := range b {
					for keyd, d := range c {
						if keya == "cash" && keyd == "value_increase" {
							fmt.Fprintln(r, keya, "\t", keyb, "\t", keyc, "\t", keyd, "\t", d)
						}
					}
				}
			}
		}
	}
	r.Flush()
	// point := Managerial_Accounting{
	// 	cvp: []cvp{
	// 		{"fe", 4, 4000, 1000, 0, []float64{4, 5}},
	// 		{"al", 4, 4000, 1800, 0, []float64{4, 5}}},
	// 	overhead: []overhead{
	// 		{"units", []float64{50}},
	// 		{"portions", []float64{10000, 5000}},
	// 		{"fixed_cost", []float64{500}},
	// 	},
	// }
	// j := point.cost_volume_profit_slice()
	// q := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// fmt.Fprintln(q, "name\t", "units\t", "selling_price_per_unit\t", "variable_cost_per_unit\t", "fixed_cost\t", "mixed_cost\t", "mixed_cost_per_unit\t", "sales\t", "profit\t", "profit_per_unit\t", "contribution_margin_per_unit\t", "contribution_margin\t", "contribution_margin_ratio\t", "break_even_in_unit\t", "break_even_in_sales\t", "degree_of_operating_leverage\t")
	// for _, i := range j {
	// 	fmt.Fprintln(q, i.name, "\t", i.units, "\t", i.selling_price_per_unit, "\t", i.variable_cost_per_unit, "\t", i.fixed_cost, "\t", i.mixed_cost, "\t", i.mixed_cost_per_unit, "\t", i.sales, "\t", i.profit, "\t", i.profit_per_unit, "\t", i.contribution_margin_per_unit, "\t", i.contribution_margin, "\t", i.contribution_margin_ratio, "\t", i.break_even_in_unit, "\t", i.break_even_in_sales, "\t", i.degree_of_operating_leverage, "\t")
	// }
	// q.Flush()
}
