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
	is_credit                    bool
	cost_flow_type, father, name string
}

type Financial_accounting struct {
	date_layout                               []string
	DriverName, DataSourceName, Database_name string
	assets                                    string
	current_assets                            string
	cash_and_cash_equivalents                 string
	short_term_investments                    string
	receivables                               string
	inventory                                 string
	liabilities                               string
	current_liabilities                       string
	equity                                    string
	retained_earnings                         string
	dividends                                 string
	income_statement                          string
	ebitda                                    string
	sales                                     string
	cost_of_goods_sold                        string
	discounts                                 string
	invoice_discount                          string
	interest_expense                          string
	accounts                                  []account
	Invoice_discounts_list                    [][2]float64
	auto_complete_entries                     [][]account_method_value_price
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
	value_ending_balance, value_beginning_balance, value_average, value_increase, value_decrease, value_increase_or_decrease, value_inflow,
	value_outflow, value_flow, value_growth_ratio, value_turnover, value_percent, value_change_since_base_period, value_growth_ratio_to_base_period,
	price_ending_balance, price_beginning_balance, price_average, price_increase, price_decrease, price_increase_or_decrease, price_inflow,
	price_outflow, price_flow, price_growth_ratio, price_turnover, price_percent, price_change_since_base_period, price_growth_ratio_to_base_period,
	quantity_ending_balance, quantity_beginning_balance, quantity_average, quantity_increase, quantity_decrease, quantity_increase_or_decrease, quantity_inflow,
	quantity_outflow, quantity_flow, quantity_growth_ratio, quantity_turnover, quantity_percent, quantity_change_since_base_period, quantity_growth_ratio_to_base_period float64
}

type financial_analysis struct {
	current_assets, current_liabilities, cash, short_term_investments, net_receivables, net_credit_sales,
	average_net_receivables, cost_of_goods_sold, average_inventory, net_income, net_sales, average_assets, average_equity,
	preferred_dividends, average_common_stockholders_equity, market_price_per_shares_outstanding, cash_dividends,
	total_debt, total_assets, ebitda, interest_expense, weighted_average_common_shares_outstanding float64
}

type financial_analysis_statement struct {
	current_ratio                        float64 // current_assets / current_liabilities
	acid_test                            float64 // (cash + short_term_investments + net_receivables) / current_liabilities
	receivables_turnover                 float64 // net_credit_sales / average_net_receivables
	inventory_turnover                   float64 // cost_of_goods_sold / average_inventory
	asset_turnover                       float64 // net_sales / average_assets
	profit_margin                        float64 // net_income / net_sales
	return_on_assets                     float64 // net_income / average_assets
	return_on_equity                     float64 // net_income / average_equity
	payout_ratio                         float64 // cash_dividends / net_income
	debt_to_total_assets_ratio           float64 // total_debt / total_assets
	times_interest_earned                float64 // ebitda / interest_expense
	return_on_common_stockholders_equity float64 // (net_income - preferred_dividends) / average_common_stockholders_equity
	earnings_per_share                   float64 // (net_income - preferred_dividends) / weighted_average_common_shares_outstanding
	price_earnings_ratio                 float64 // market_price_per_shares_outstanding / earnings_per_share
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
	name                                                                         string
	units, units_gap, selling_price_per_unit, variable_cost_per_unit, fixed_cost float64
	portions                                                                     []float64
}

type overhead struct {
	sales_or_variable_or_fixed, distribution_method string
	fixed_cost                                      []float64
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
	db                   *sql.DB
	inventory            []string
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

	var all_accounts []string
	for _, i := range s.accounts {
		if !s.is_father("", i.name) {
			log.Panic(i.name, " account does not ends in ''")
		}
		all_accounts = append(all_accounts, i.name)
		switch {
		case is_in(i.cost_flow_type, []string{"fifo", "lifo", "wma"}) && !s.is_father(s.retained_earnings, i.name) && !i.is_credit:
			inventory = append(inventory, i.name)
		case i.cost_flow_type == "":
		default:
			log.Panic(i.cost_flow_type, " for ", i.name, " is not in [fifo,lifo,wma,''] or you can't use it with ", s.retained_earnings, " or is_credit==true")
		}
	}

	switch {
	case !s.is_father(s.assets, s.current_assets):
		log.Panic(s.assets, " should be one of the fathers of ", s.current_assets)
	case !s.is_father(s.current_assets, s.cash_and_cash_equivalents):
		log.Panic(s.current_assets, " should be one of the fathers of ", s.cash_and_cash_equivalents)
	case !s.is_father(s.current_assets, s.short_term_investments):
		log.Panic(s.current_assets, " should be one of the fathers of ", s.short_term_investments)
	case !s.is_father(s.current_assets, s.receivables):
		log.Panic(s.current_assets, " should be one of the fathers of ", s.receivables)
	case !s.is_father(s.current_assets, s.inventory):
		log.Panic(s.current_assets, " should be one of the fathers of ", s.inventory)
	case !s.is_father(s.liabilities, s.current_liabilities):
		log.Panic(s.liabilities, " should be one of the fathers of ", s.current_liabilities)
	case !s.is_father(s.equity, s.retained_earnings):
		log.Panic(s.equity, " should be one of the fathers of ", s.retained_earnings)
	case !s.is_father(s.retained_earnings, s.dividends):
		log.Panic(s.retained_earnings, " should be one of the fathers of ", s.dividends)
	case !s.is_father(s.retained_earnings, s.income_statement):
		log.Panic(s.retained_earnings, " should be one of the fathers of ", s.income_statement)
	case !s.is_father(s.income_statement, s.ebitda):
		log.Panic(s.income_statement, " should be one of the fathers of ", s.ebitda)
	case !s.is_father(s.income_statement, s.interest_expense):
		log.Panic(s.income_statement, " should be one of the fathers of ", s.interest_expense)
	case !s.is_father(s.ebitda, s.sales):
		log.Panic(s.ebitda, " should be one of the fathers of ", s.sales)
	case !s.is_father(s.ebitda, s.cost_of_goods_sold):
		log.Panic(s.ebitda, " should be one of the fathers of ", s.cost_of_goods_sold)
	case !s.is_father(s.ebitda, s.discounts):
		log.Panic(s.ebitda, " should be one of the fathers of ", s.discounts)
	case !s.is_father(s.discounts, s.invoice_discount):
		log.Panic(s.discounts, " should be one of the fathers of ", s.invoice_discount)
	}
	check_if_duplicates(all_accounts)
	check_accounts("account", "inventory", " is not have fifo lifo wma on cost_flow_type field", inventory)

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
			s.check_debit_equal_credit(double_entry)
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
			if s.is_father(s.income_statement, entry.Account) && s.is_credit(entry.Account) {
				total_invoice_before_invoice_discount += entry.value
			} else if s.is_father(s.discounts, entry.Account) && !s.is_credit(entry.Account) {
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
		if !(s.is_father(s.equity, entry.Account) && s.is_credit(entry.Account)) {
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

	s.check_debit_equal_credit(array_of_entry)

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

func (s Financial_accounting) financial_statements(start_date, end_date time.Time, periods int, names []string) ([][]statement, []financial_analysis_statement, []map[string]map[string]map[string]map[string]map[string]float64, []journal_tag) {
	check_dates(start_date, end_date)
	days := int(end_date.Sub(start_date).Hours() / 24)
	var journal []journal_tag
	rows, _ := db.Query("select * from journal order by date,entry_number")
	for rows.Next() {
		var entry journal_tag
		rows.Scan(&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.price, &entry.quantity, &entry.barcode, &entry.entry_expair, &entry.description, &entry.name, &entry.employee_name, &entry.entry_date, &entry.reverse)
		journal = append(journal, entry)
	}
	all_values_for_all := []map[string]map[string]map[string]map[string]map[string]float64{}
	for a := 0; a < periods; a++ {
		all_values_for_all = append(all_values_for_all, s.all_values(journal, start_date.AddDate(0, 0, -days*a), end_date.AddDate(0, 0, -days*a), float64(days)))
	}
	var all_analysis []financial_analysis_statement
	var statements [][]statement
	for _, a := range all_values_for_all {
		statement, analysis := s.prepare_statement(a, all_values_for_all[periods-1], names)
		statements = append(statements, statement)
		all_analysis = append(all_analysis, analysis)
	}
	return statements, all_analysis, all_values_for_all, journal
}

func (s Financial_accounting) invoice(array_of_journal_tag []journal_tag) []invoice_struct {
	m := map[string]*invoice_struct{}
	for _, entry := range array_of_journal_tag {
		var key string
		switch {
		case s.is_father(s.assets, entry.account) && !s.is_credit(entry.account) && !is_in(entry.account, inventory) && entry.value > 0:
			key = "total"
		case s.is_father(s.discounts, entry.account) && !s.is_credit(entry.account):
			key = "total discounts"
		case s.is_father(s.sales, entry.account) && s.is_credit(entry.account):
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
	case s.return_cost_flow_type(account) == "fifo":
		order_by_date_asc_or_desc = "asc"
	case s.return_cost_flow_type(account) == "lifo":
		order_by_date_asc_or_desc = "desc"
	case s.return_cost_flow_type(account) == "wma":
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

func (s Financial_accounting) all_values(journal []journal_tag, start_date, end_date time.Time, days float64) map[string]map[string]map[string]map[string]map[string]float64 {
	var one_compound_entry []journal_tag
	var previous_date string
	var previous_entry_number int
	var date time.Time
	all_flows := map[string]map[string]map[string]map[string]map[string]float64{}
	all_nan_flow := map[string]map[string]map[string]map[string]float64{}
	for _, entry := range journal {
		date = s.parse_date(entry.date)
		if previous_date != entry.date || previous_entry_number != entry.entry_number {
			s.sum_flow(date, start_date, one_compound_entry, all_flows)
			s.sum_values(date, start_date, one_compound_entry, all_nan_flow)
			one_compound_entry = []journal_tag{}
		}
		if date.Before(end_date) {
			one_compound_entry = append(one_compound_entry, entry)
		}
		previous_date = entry.date
		previous_entry_number = entry.entry_number
	}
	s.sum_flow(date, start_date, one_compound_entry, all_flows)
	s.sum_values(date, start_date, one_compound_entry, all_nan_flow)

	all_values1 := map[string]map[string]map[string]map[string]map[string]float64{}
	for key_account_flow, _ := range all_nan_flow {
		if all_values1[key_account_flow] == nil {
			all_values1[key_account_flow] = map[string]map[string]map[string]map[string]float64{}
		}
		for key_account, map_account := range all_nan_flow {
			if all_values1[key_account_flow][key_account] == nil {
				all_values1[key_account_flow][key_account] = map[string]map[string]map[string]float64{}
			}
			for key_name, map_name := range map_account {
				if all_values1[key_account_flow][key_account][key_name] == nil {
					all_values1[key_account_flow][key_account][key_name] = map[string]map[string]float64{}
				}
				for key_vpq, map_vpq := range map_name {
					if all_values1[key_account_flow][key_account][key_name][key_vpq] == nil {
						all_values1[key_account_flow][key_account][key_name][key_vpq] = map[string]float64{}
					}
					for key_number, _ := range map_vpq {
						all_values1[key_account_flow][key_account][key_name][key_vpq][key_number] = map_vpq[key_number]
						all_values1[key_account_flow][key_account][key_name][key_vpq]["inflow"] = all_flows[key_account_flow][key_account][key_name][key_vpq]["inflow"]
						all_values1[key_account_flow][key_account][key_name][key_vpq]["outflow"] = all_flows[key_account_flow][key_account][key_name][key_vpq]["outflow"]
					}
				}
			}
		}
	}

	new_all_values := map[string]map[string]map[string]map[string]map[string]float64{}
	for key_account_flow, map_account_flow := range all_values1 {
		if new_all_values[key_account_flow] == nil {
			new_all_values[key_account_flow] = map[string]map[string]map[string]map[string]float64{}
		}
		for key_account, map_account := range map_account_flow {
			var last_name string
			key1 := key_account
			for {
				for _, ss := range s.accounts {
					if ss.name == key_account {
						key_account = ss.father
						if new_all_values[key_account_flow][ss.name] == nil {
							new_all_values[key_account_flow][ss.name] = map[string]map[string]map[string]float64{}
						}
						for key_name, map_name := range map_account {
							if new_all_values[key_account_flow][ss.name][key_name] == nil {
								new_all_values[key_account_flow][ss.name][key_name] = map[string]map[string]float64{}
							}
							for key_vpq, map_vpq := range map_name {
								if new_all_values[key_account_flow][ss.name][key_name][key_vpq] == nil {
									new_all_values[key_account_flow][ss.name][key_name][key_vpq] = map[string]float64{}
								}
								for key_number, number := range map_vpq {
									switch {
									case !is_in(key_number, []string{"inflow", "outflow"}):
										if s.is_credit(key1) == s.is_credit(ss.name) {
											new_all_values[key_account_flow][ss.name][key_name][key_vpq][key_number] += number
										} else {
											new_all_values[key_account_flow][ss.name][key_name][key_vpq][key_number] -= number
										}
									case key_account_flow != key1:
										new_all_values[key_account_flow][ss.name][key_name][key_vpq][key_number] += number
									case key_account_flow == ss.name:
										new_all_values[key_account_flow][key1][key_name][key_vpq][key_number] += number
									}
								}
							}
						}
					}
				}
				if last_name == key_account {
					break
				}
				last_name = key_account
			}
		}
	}
	for key_account_flow, map_account_flow := range new_all_values {
		for _, map_account := range map_account_flow {
			for key_name, map_name := range map_account {
				if map_name["price"] == nil {
					map_name["price"] = map[string]float64{}
				}
				for key_vpq, map_vpq := range map_name {
					map_vpq["increase_or_decrease"] = map_vpq["increase"] - map_vpq["decrease"]
					map_vpq["ending_balance"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]
					map_vpq["flow"] = map_vpq["inflow"] - map_vpq["outflow"]
					map_vpq["average"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]/2
					map_vpq["turnover"] = map_vpq["inflow"] / map_vpq["average"]
					map_vpq["turnover_days"] = days / map_vpq["turnover"]
					map_vpq["growth_ratio"] = map_vpq["ending_balance"] / map_vpq["beginning_balance"]
					map_vpq["percent"] = map_vpq["ending_balance"] /
						(map_account_flow[key_account_flow][key_name][key_vpq]["beginning_balance"] + map_account_flow[key_account_flow][key_name][key_vpq]["increase"] - map_account_flow[key_account_flow][key_name][key_vpq]["decrease"])
					for key_number, _ := range map_vpq {
						map_name["price"][key_number] = map_name["value"][key_number] / map_name["quantity"][key_number]
					}
				}
			}
		}
	}
	return new_all_values
}

func (s Financial_accounting) sum_values(date, start_date time.Time, one_compound_entry []journal_tag, all_flows map[string]map[string]map[string]map[string]float64) {
	for _, b := range one_compound_entry {
		if all_flows[b.account] == nil {
			all_flows[b.account] = map[string]map[string]map[string]float64{}
		}
		if all_flows[b.account][b.name] == nil {
			all_flows[b.account][b.name] = map[string]map[string]float64{}
		}
		if all_flows[b.account][b.name]["value"] == nil {
			all_flows[b.account][b.name]["value"] = map[string]float64{}
		}
		if all_flows[b.account][b.name]["quantity"] == nil {
			all_flows[b.account][b.name]["quantity"] = map[string]float64{}
		}
		if all_flows[s.retained_earnings] == nil {
			all_flows[s.retained_earnings] = map[string]map[string]map[string]float64{}
		}
		if all_flows[s.retained_earnings][b.name] == nil {
			all_flows[s.retained_earnings][b.name] = map[string]map[string]float64{}
		}
		if all_flows[s.retained_earnings][b.name]["value"] == nil {
			all_flows[s.retained_earnings][b.name]["value"] = map[string]float64{}
		}
		if all_flows[s.retained_earnings][b.name]["quantity"] == nil {
			all_flows[s.retained_earnings][b.name]["quantity"] = map[string]float64{}
		}
		if date.Before(start_date) {
			switch {
			case s.is_father(s.retained_earnings, b.account) && s.is_credit(b.account):
				all_flows[s.retained_earnings][b.name]["value"]["beginning_balance"] += b.value
				all_flows[s.retained_earnings][b.name]["quantity"]["beginning_balance"] += b.quantity
			case s.is_father(s.retained_earnings, b.account) && !s.is_credit(b.account):
				all_flows[s.retained_earnings][b.name]["value"]["beginning_balance"] -= b.value
				all_flows[s.retained_earnings][b.name]["quantity"]["beginning_balance"] -= b.quantity
			default:
				all_flows[b.account][b.name]["value"]["beginning_balance"] += b.value
				all_flows[b.account][b.name]["quantity"]["beginning_balance"] += b.quantity
			}
		}
		if date.After(start_date) {
			if b.value >= 0 {
				all_flows[b.account][b.name]["value"]["increase"] += math.Abs(b.value)
				all_flows[b.account][b.name]["quantity"]["increase"] += math.Abs(b.quantity)
			} else {
				all_flows[b.account][b.name]["value"]["decrease"] += math.Abs(b.value)
				all_flows[b.account][b.name]["quantity"]["decrease"] += math.Abs(b.quantity)
			}
		}
	}
}

func (s Financial_accounting) sum_flow(date, start_date time.Time, one_compound_entry []journal_tag, all_flows map[string]map[string]map[string]map[string]map[string]float64) {
	for _, a := range one_compound_entry {
		if all_flows[a.account] == nil {
			all_flows[a.account] = map[string]map[string]map[string]map[string]float64{}
		}
		for _, b := range one_compound_entry {
			if all_flows[a.account][b.account] == nil {
				all_flows[a.account][b.account] = map[string]map[string]map[string]float64{}
			}
			if all_flows[a.account][b.account][b.name] == nil {
				all_flows[a.account][b.account][b.name] = map[string]map[string]float64{}
			}
			if all_flows[a.account][b.account][b.name]["value"] == nil {
				all_flows[a.account][b.account][b.name]["value"] = map[string]float64{}
			}
			if all_flows[a.account][b.account][b.name]["quantity"] == nil {
				all_flows[a.account][b.account][b.name]["quantity"] = map[string]float64{}
			}
			if date.After(start_date) {
				if b.account == a.account || s.is_credit(b.account) != s.is_credit(a.account) {
					sum_flows(a, b, 1, all_flows)
				} else {
					sum_flows(a, b, -1, all_flows)
				}
			}
		}
	}
}

func (s Financial_accounting) prepare_statement(statement_map, statement_map_base map[string]map[string]map[string]map[string]map[string]float64, names []string) ([]statement, financial_analysis_statement) {
	new_statement_map_base := s.sum_and_extract_new_map(statement_map_base, names, s.cash_and_cash_equivalents)
	new_statement_map_cash := s.sum_and_extract_new_map(statement_map, names, s.cash_and_cash_equivalents)
	new_statement_map_sales := s.sum_and_extract_new_map(statement_map, names, s.sales)
	new_statement_map_cost_of_goods_sold := s.sum_and_extract_new_map(statement_map, names, s.cost_of_goods_sold)

	analysis := financial_analysis{
		current_assets:                      new_statement_map_cash[s.current_assets]["value"]["ending_balance"],
		current_liabilities:                 new_statement_map_cash[s.current_liabilities]["value"]["ending_balance"],
		cash:                                new_statement_map_cash[s.cash_and_cash_equivalents]["value"]["ending_balance"],
		short_term_investments:              new_statement_map_cash[s.short_term_investments]["value"]["ending_balance"],
		net_receivables:                     new_statement_map_cash[s.receivables]["value"]["ending_balance"],
		net_credit_sales:                    new_statement_map_sales[s.receivables]["value"]["flow"],
		average_net_receivables:             new_statement_map_cash[s.receivables]["value"]["average"],
		cost_of_goods_sold:                  new_statement_map_cash[s.cost_of_goods_sold]["value"]["ending_balance"],
		average_inventory:                   new_statement_map_cash[s.inventory]["value"]["average"],
		net_income:                          new_statement_map_cash[s.income_statement]["value"]["ending_balance"],
		net_sales:                           new_statement_map_cash[s.sales]["value"]["ending_balance"],
		average_assets:                      new_statement_map_cash[s.assets]["value"]["average"],
		average_equity:                      new_statement_map_cash[s.equity]["value"]["average"],
		preferred_dividends:                 0,
		average_common_stockholders_equity:  0,
		market_price_per_shares_outstanding: 0,
		cash_dividends:                      new_statement_map_cash[s.dividends]["value"]["flow"],
		total_debt:                          new_statement_map_cash[s.liabilities]["value"]["ending_balance"],
		total_assets:                        new_statement_map_cash[s.assets]["value"]["ending_balance"],
		ebitda:                              new_statement_map_cash[s.ebitda]["value"]["ending_balance"],
		interest_expense:                    new_statement_map_cash[s.interest_expense]["value"]["ending_balance"],
		weighted_average_common_shares_outstanding: 0,
	}.financial_analysis_statement()

	value_total_assets := new_statement_map_cash[s.assets]["value"]["ending_balance"]
	price_total_assets := new_statement_map_cash[s.assets]["price"]["ending_balance"]
	quantity_total_assets := new_statement_map_cash[s.assets]["quantity"]["ending_balance"]
	value_total_sales := new_statement_map_cash[s.sales]["value"]["ending_balance"]
	price_total_sales := new_statement_map_cash[s.sales]["price"]["ending_balance"]
	quantity_total_sales := new_statement_map_cash[s.sales]["quantity"]["ending_balance"]

	var value_total, price_total, quantity_total float64
	var value_turnover, price_turnover, quantity_turnover float64
	var statement_sheet []statement

	for key_account, map_account := range new_statement_map_cash {
		var okeys []bool
		for _, map_vpq := range map_account {
			for _, number := range map_vpq {
				if number == 0 {
					okeys = append(okeys, false)
				} else if math.IsNaN(number) {
					okeys = append(okeys, false)
				} else {
					okeys = append(okeys, true)
				}
			}
		}
		var ok bool
		for _, a := range okeys {
			if a {
				ok = true
			}
		}
		if !ok {
			continue
		}

		if !s.is_father(s.income_statement, key_account) {
			value_total = value_total_assets
			price_total = price_total_assets
			quantity_total = quantity_total_assets
		} else {
			value_total = value_total_sales
			price_total = price_total_sales
			quantity_total = quantity_total_sales
		}
		switch {
		case s.is_father(s.inventory, key_account):
			value_turnover = new_statement_map_cost_of_goods_sold[key_account]["value"]["turnover"]
			price_turnover = new_statement_map_cost_of_goods_sold[key_account]["price"]["turnover"]
			quantity_turnover = new_statement_map_cost_of_goods_sold[key_account]["quantity"]["turnover"]
		case s.is_father(s.assets, key_account):
			value_turnover = new_statement_map_sales[key_account]["value"]["turnover"]
			price_turnover = new_statement_map_sales[key_account]["price"]["turnover"]
			quantity_turnover = new_statement_map_sales[key_account]["quantity"]["turnover"]
		default:
			value_turnover = new_statement_map_cash[key_account]["value"]["turnover"]
			price_turnover = new_statement_map_cash[key_account]["price"]["turnover"]
			quantity_turnover = new_statement_map_cash[key_account]["quantity"]["turnover"]
		}

		statement_sheet = append(statement_sheet, statement{
			account:                              key_account,
			value_ending_balance:                 map_account["value"]["ending_balance"],
			value_beginning_balance:              map_account["value"]["beginning_balance"],
			value_average:                        map_account["value"]["average"],
			value_increase:                       map_account["value"]["increase"],
			value_decrease:                       map_account["value"]["decrease"],
			value_increase_or_decrease:           map_account["value"]["increase_or_decrease"],
			value_inflow:                         map_account["value"]["inflow"],
			value_outflow:                        map_account["value"]["outflow"],
			value_flow:                           map_account["value"]["flow"],
			value_growth_ratio:                   map_account["value"]["growth_ratio"],
			value_turnover:                       value_turnover,
			value_percent:                        map_account["value"]["ending_balance"] / value_total,
			value_change_since_base_period:       map_account["value"]["ending_balance"] - new_statement_map_base[key_account]["value"]["ending_balance"],
			value_growth_ratio_to_base_period:    map_account["value"]["ending_balance"] / new_statement_map_base[key_account]["value"]["ending_balance"],
			price_ending_balance:                 map_account["price"]["ending_balance"],
			price_beginning_balance:              map_account["price"]["beginning_balance"],
			price_average:                        map_account["price"]["average"],
			price_increase:                       map_account["price"]["increase"],
			price_decrease:                       map_account["price"]["decrease"],
			price_increase_or_decrease:           map_account["price"]["increase_or_decrease"],
			price_inflow:                         map_account["price"]["inflow"],
			price_outflow:                        map_account["price"]["outflow"],
			price_flow:                           map_account["price"]["flow"],
			price_growth_ratio:                   map_account["price"]["growth_ratio"],
			price_turnover:                       price_turnover,
			price_percent:                        map_account["price"]["ending_balance"] / price_total,
			price_change_since_base_period:       map_account["price"]["ending_balance"] - new_statement_map_base[key_account]["price"]["ending_balance"],
			price_growth_ratio_to_base_period:    map_account["price"]["ending_balance"] / new_statement_map_base[key_account]["price"]["ending_balance"],
			quantity_ending_balance:              map_account["quantity"]["ending_balance"],
			quantity_beginning_balance:           map_account["quantity"]["beginning_balance"],
			quantity_average:                     map_account["quantity"]["average"],
			quantity_increase:                    map_account["quantity"]["increase"],
			quantity_decrease:                    map_account["quantity"]["decrease"],
			quantity_increase_or_decrease:        map_account["quantity"]["increase_or_decrease"],
			quantity_inflow:                      map_account["quantity"]["inflow"],
			quantity_outflow:                     map_account["quantity"]["outflow"],
			quantity_flow:                        map_account["quantity"]["flow"],
			quantity_growth_ratio:                map_account["quantity"]["growth_ratio"],
			quantity_turnover:                    quantity_turnover,
			quantity_percent:                     map_account["quantity"]["ending_balance"] / quantity_total,
			quantity_change_since_base_period:    map_account["quantity"]["ending_balance"] - new_statement_map_base[key_account]["quantity"]["ending_balance"],
			quantity_growth_ratio_to_base_period: map_account["quantity"]["ending_balance"] / new_statement_map_base[key_account]["quantity"]["ending_balance"],
		})
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
	return statement_sheet, analysis
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

func (s Financial_accounting) is_father(father, name string) bool {
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

func (s Financial_accounting) return_cost_flow_type(name string) string {
	for _, a := range s.accounts {
		if a.name == name {
			return a.cost_flow_type
		}
	}
	return ""
}

func (s Financial_accounting) is_credit(name string) bool {
	for _, a := range s.accounts {
		if a.name == name {
			return a.is_credit
		}
	}
	log.Panic(name, " is not debit nor credit")
	return false
}

func (s Financial_accounting) check_debit_equal_credit(array_of_entry []Account_value_quantity_barcode) {
	var zero float64
	var debit_number, credit_number int
	for _, entry := range array_of_entry {
		switch s.is_credit(entry.Account) {
		case false:
			zero += entry.value
			if entry.value >= 0 {
				debit_number++
			} else {
				credit_number++
			}
		case true:
			zero -= entry.value
			if entry.value <= 0 {
				debit_number++
			} else {
				credit_number++
			}
		}
	}
	if (debit_number != 1) && (credit_number != 1) {
		// log.Panic("should be one credit or one debit in the entry")
	}
	if zero != 0 {
		log.Panic(zero, " not equal 0 if the number>0 it means debit overstated else credit overstated debit-credit should equal zero ", array_of_entry)
	}
}

func (s Financial_accounting) sum_and_extract_new_map(statement_map map[string]map[string]map[string]map[string]map[string]float64, name []string, flow_account string) map[string]map[string]map[string]float64 {
	var flow_accounts []string
	for _, a := range s.accounts {
		if s.is_father(flow_account, a.name) {
			flow_accounts = append(flow_accounts, a.name)
		}
	}

	new_statement_map := map[string]map[string]map[string]float64{}
	for key_account_flow, map_account_flow := range statement_map {
		if is_in(key_account_flow, flow_accounts) {
			for key_account, map_account := range map_account_flow {
				if new_statement_map[key_account] == nil {
					new_statement_map[key_account] = map[string]map[string]float64{}
				}
				for key_name, map_name := range map_account {
					var ok bool
					if len(name) == 0 {
						ok = true
					} else if is_in(key_name, name) {
						ok = true
					}
					if ok {
						for key_vpq, map_vpq := range map_name {
							if new_statement_map[key_account][key_vpq] == nil {
								new_statement_map[key_account][key_vpq] = map[string]float64{}
							}
							for key_number, number := range map_vpq {
								switch {
								case is_in(key_number, []string{"inflow", "outflow"}):
									if s.is_credit(flow_account) == s.is_credit(key_account_flow) {
										new_statement_map[key_account][key_vpq][key_number] += number
									} else {
										new_statement_map[key_account][key_vpq][key_number] -= number
									}
								default:
									new_statement_map[key_account][key_vpq][key_number] += number
								}
							}
						}
					}
				}
			}
		}
	}
	for _, map_account := range new_statement_map {
		if map_account["price"] == nil {
			map_account["price"] = map[string]float64{}
		}
		for key_vpq, map_vpq := range map_account {
			map_vpq["increase_or_decrease"] = map_vpq["increase"] - map_vpq["decrease"]
			map_vpq["ending_balance"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]
			map_vpq["flow"] = map_vpq["inflow"] - map_vpq["outflow"]
			map_vpq["average"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]/2
			map_vpq["turnover"] = map_vpq["inflow"] / map_vpq["average"]
			map_vpq["growth_ratio"] = map_vpq["ending_balance"] / map_vpq["beginning_balance"]
			map_vpq["percent"] = map_vpq["ending_balance"] / (new_statement_map[flow_account][key_vpq]["beginning_balance"] + new_statement_map[flow_account][key_vpq]["increase"] - new_statement_map[flow_account][key_vpq]["decrease"])
			for key_number, _ := range map_vpq {
				map_account["price"][key_number] = map_account["value"][key_number] / map_account["quantity"][key_number]
			}
		}
	}
	return new_statement_map
}

func sum_flows(a journal_tag, b journal_tag, x float64, all_flows map[string]map[string]map[string]map[string]map[string]float64) {
	if b.value*x < 0 {
		all_flows[a.account][b.account][b.name]["value"]["outflow"] += math.Abs(b.value)
		all_flows[a.account][b.account][b.name]["quantity"]["outflow"] += math.Abs(b.quantity)
	} else {
		all_flows[a.account][b.account][b.name]["value"]["inflow"] += math.Abs(b.value)
		all_flows[a.account][b.account][b.name]["quantity"]["inflow"] += math.Abs(b.quantity)
	}
}

// func calculate_vpq(new_all_values map[string]map[string]map[string]map[string]map[string]float64) {
// 	for _, map_account_flow := range new_all_values {
// 		for key_account, map_account := range map_account_flow {
// 			for key_name, map_name := range map_account {
// 				if map_name["price"] == nil {
// 					map_name["price"] = map[string]float64{}
// 				}
// 				for key_vpq, map_vpq := range map_name {
// 					map_vpq["increase_or_decrease"] = map_vpq["increase"] - map_vpq["decrease"]
// 					map_vpq["ending_balance"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]
// 					map_vpq["flow"] = map_vpq["inflow"] - map_vpq["outflow"]
// 					map_vpq["average"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]/2
// 					map_vpq["turnover"] = map_vpq["inflow"] / map_vpq["average"]
// 					map_vpq["growth_ratio"] = map_vpq["ending_balance"] / map_vpq["beginning_balance"]
// 					map_vpq["percent"] = map_account_flow[key_account][key_name][key_vpq]["ending_balance"] / (map_account_flow[key_account][key_name][key_vpq]["beginning_balance"] + map_account_flow[key_account][key_name][key_vpq]["beginning_balance"] - map_account_flow[key_account][key_name][key_vpq]["beginning_balance"])
// 					for key_number, _ := range map_vpq {
// 						map_name["price"][key_number] = map_name["value"][key_number] / map_name["quantity"][key_number]
// 					}
// 				}
// 			}
// 		}
// 	}
// }

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
	return_on_equity := s.net_income / s.average_equity
	payout_ratio := s.cash_dividends / s.net_income
	debt_to_total_assets_ratio := s.total_debt / s.total_assets
	times_interest_earned := s.ebitda / s.interest_expense
	return_on_common_stockholders_equity := (s.net_income - s.preferred_dividends) / s.average_common_stockholders_equity
	earnings_per_share := (s.net_income - s.preferred_dividends) / s.weighted_average_common_shares_outstanding
	price_earnings_ratio := s.market_price_per_shares_outstanding / earnings_per_share
	return financial_analysis_statement{
		current_ratio:                        current_ratio,
		acid_test:                            acid_test,
		receivables_turnover:                 receivables_turnover,
		inventory_turnover:                   inventory_turnover,
		profit_margin:                        profit_margin,
		asset_turnover:                       asset_turnover,
		return_on_assets:                     return_on_assets,
		return_on_equity:                     return_on_equity,
		payout_ratio:                         payout_ratio,
		debt_to_total_assets_ratio:           debt_to_total_assets_ratio,
		times_interest_earned:                times_interest_earned,
		return_on_common_stockholders_equity: return_on_common_stockholders_equity,
		earnings_per_share:                   earnings_per_share,
		price_earnings_ratio:                 price_earnings_ratio,
	}
}

func (s Managerial_Accounting) decrease() float64 {
	return s.beginning_balance + s.increase - s.ending_balance
}

func (s Managerial_Accounting) cost_of_goods_sold() float64 {
	return s.decrease() - s.decreases_in_account_caused_by_not_sell
}

func (s Managerial_Accounting) cost_volume_profit_slice() []cvp_statistics {
	length_fixed_cost := len(s.cvp[0].portions)
	total_portions := make([]float64, length_fixed_cost)
	for _, i := range s.cvp {
		if length_fixed_cost != len(i.portions) {
			log.Panic("length of portions and fixed_cost in overhead that have portions distribution_method, should be all the same length")
		}
		for index, i := range i.portions {
			total_portions[index] += i
		}
	}
	for _, i := range s.overhead {
		if length_fixed_cost != len(i.fixed_cost) && i.distribution_method == "portions" {
			log.Panic("length of portions and fixed_cost in overhead that have portions distribution_method, should be all the same length")
		}
	}
	var h []cvp_statistics
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
			var total_overhead_cost_to_sum float64
			switch a.distribution_method {
			case "percent_from_sales":
				total_overhead_cost_to_sum = total_overhead_cost * b.sales
			case "units_gap":
				total_overhead_cost_to_sum = s.cvp[indexb].units_gap * b.variable_cost_per_unit
				b.units -= s.cvp[indexb].units_gap
			case "1":
				total_overhead_cost_to_sum = total_overhead_cost
			case "equally":
				total_overhead_cost_to_sum = total_overhead_cost / float64(len(s.cvp))
			case "units":
				total_overhead_cost_to_sum = total_overhead_cost * b.units / totals.units
			case "fixed_cost":
				total_overhead_cost_to_sum = total_overhead_cost * b.fixed_cost / totals.fixed_cost
			case "mixed_cost":
				total_overhead_cost_to_sum = total_overhead_cost * b.mixed_cost / totals.mixed_cost
			case "sales":
				total_overhead_cost_to_sum = total_overhead_cost * b.sales / totals.sales
			case "profit":
				total_overhead_cost_to_sum = total_overhead_cost * b.profit / totals.profit
			case "contribution_margin":
				total_overhead_cost_to_sum = total_overhead_cost * b.contribution_margin / totals.contribution_margin
			case "portions":
				var sum_portions_cost float64
				for indexc, c := range s.cvp[indexb].portions {
					sum_portions_cost += c / total_portions[indexc] * a.fixed_cost[indexc]
				}
				total_overhead_cost_to_sum = total_overhead_cost * sum_portions_cost / total_overhead_cost
			default:
				log.Panic(a.distribution_method, " is not in [percent_from_sales,units_gap,1,equally,units,fixed_cost,mixed_cost,sales,profit,contribution_margin,portions]")
			}
			switch a.sales_or_variable_or_fixed {
			case "sales":
				b.selling_price_per_unit = ((b.selling_price_per_unit * b.units) - total_overhead_cost_to_sum) / b.units
			case "variable":
				b.variable_cost_per_unit = ((b.variable_cost_per_unit * b.units) + total_overhead_cost_to_sum) / b.units
			case "fixed":
				b.fixed_cost += total_overhead_cost_to_sum
			default:
				log.Panic(a.sales_or_variable_or_fixed, " is not in [sales,variable,fixed]")
			}
			h[indexb] = cost_volume_profit(b.name, b.units, b.selling_price_per_unit, b.variable_cost_per_unit, b.fixed_cost)
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
		date_layout:               []string{"2006-01-02 15:04:05.999999999 -0700 +03 m=+0.999999999", "2006-01-02 15:04:05.999999999 -0700 +03"},
		DriverName:                "mysql",
		DataSourceName:            "hashem:hashem@tcp(localhost)/",
		Database_name:             "acc",
		assets:                    "assets",
		current_assets:            "current_assets",
		cash_and_cash_equivalents: "cash_and_cash_equivalents",
		short_term_investments:    "short_term_investments",
		receivables:               "receivables",
		inventory:                 "inventory",
		liabilities:               "liabilities",
		current_liabilities:       "current_liabilities",
		equity:                    "equity",
		retained_earnings:         "retained_earnings",
		dividends:                 "dividends",
		income_statement:          "income_statement",
		ebitda:                    "ebitda",
		sales:                     "sales",
		cost_of_goods_sold:        "cost_of_goods_sold",
		discounts:                 "discounts",
		invoice_discount:          "invoice_discount",
		interest_expense:          "interest_expense",
		accounts: []account{
			{false, "wma", "", "assets"},
			{false, "wma", "assets", "current_assets"},
			{false, "wma", "current_assets", "cash_and_cash_equivalents"},
			{false, "wma", "cash_and_cash_equivalents", "cash"},
			{false, "wma", "current_assets", "short_term_investments"},
			{false, "", "current_assets", "receivables"},
			{false, "wma", "current_assets", "inventory"},
			{false, "wma", "inventory", "book"},
			{true, "", "", "liabilities"},
			{true, "", "liabilities", "current_liabilities"},
			{true, "", "current_liabilities", "tax"},
			{true, "", "", "equity"},
			{true, "", "equity", "retained_earnings"},
			{true, "", "retained_earnings", "dividends"},
			{true, "", "retained_earnings", "income_statement"},
			{true, "", "income_statement", "Revenues"},
			{true, "", "income_statement", "ebitda"},
			{true, "", "ebitda", "sales"},
			{true, "", "sales", "service revenue"},
			{true, "", "sales", "revenue of book"},
			{false, "", "ebitda", "expair_expenses"},
			{false, "", "ebitda", "cost_of_goods_sold"},
			{false, "", "cost_of_goods_sold", "cost of book"},
			{false, "", "ebitda", "discounts"},
			{false, "", "discounts", "discount of book"},
			{false, "", "discounts", "invoice_discount"},
			{false, "", "discounts", "service_discount"},
			{false, "", "income_statement", "expenses"},
			{false, "", "expenses", "interest_expense"},
			{false, "", "expenses", "tax of book"},
			{false, "", "expenses", "tax of service revenue"},
			{false, "", "expenses", "invoice_tax"}},
		Invoice_discounts_list: [][2]float64{{5, -10}},
		auto_complete_entries: [][]account_method_value_price{{{"service revenue", "quantity_ratio", 0, 10}, {"tax of service revenue", "value", 1, 1}, {"tax", "value", 1, 1}, {"service_discount", "value", 1, 1}},
			{{"book", "quantity_ratio", -1, 0}, {"revenue of book", "quantity_ratio", 1, 10}, {"cost of book", "copy_abs", 0, 0}, {"tax of book", "value", 1, 1}, {"tax", "value", 1, 1}, {"discount of book", "value", 1, 1}}},
	}
	v.initialize()
	// v.journal_entry([]Account_value_quantity_barcode{{"service revenue", 10, 100, ""},{"cash", 989, 989, ""}}, false, true, Now,
	// 	time.Time{}, "", "", "yasa", "hashem", []day_start_end{})
	// entry := select_journal(0, "cash", time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local))
	// fmt.Println(v.invoice(entry))
	// reverse_entry(2, time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local), "hashem")
	// r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// for _, i := range entry {
	// 	fmt.Fprintln(r, "\t", i.date, "\t", i.entry_number, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.barcode, "\t", i.entry_expair, "\t", i.description, "\t", i.name, "\t", i.employee_name, "\t", i.entry_date, "\t", i.reverse)
	// }
	// r.Flush()

	balance_sheet, financial_analysis_statement, all_flows_for_all, _ := v.financial_statements(
		time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local),
		time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local),
		1, []string{})
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, "index\t", "account\t",
		"value_ending_balance\t", "value_average\t", "value_increase\t", "value_decrease\t", "value_increase_or_decrease\t", "value_inflow\t",
		"value_outflow\t", "value_flow\t", "value_turnover\t", "value_percent\t", "value_change_since_base_period\t", "value_growth_ratio_to_base_period\t",
		"price_ending_balance\t", "price_average\t", "price_increase\t", "price_decrease\t", "price_increase_or_decrease\t", "price_inflow\t",
		"price_outflow\t", "price_flow\t", "price_turnover\t", "price_percent\t", "price_change_since_base_period\t", "price_growth_ratio_to_base_period\t",
		"quantity_ending_balance\t", "quantity_average\t", "quantity_increase\t", "quantity_decrease\t", "quantity_increase_or_decrease\t", "quantity_inflow\t",
		"quantity_outflow\t", "quantity_flow\t", "quantity_turnover\t", "quantity_percent\t", "quantity_change_since_base_period\t", "quantity_growth_ratio_to_base_period\t")
	for index, a := range balance_sheet {
		for _, b := range a {
			fmt.Fprintln(w, index, "\t", b.account, "\t",
				b.value_ending_balance, "\t", b.value_average, "\t", b.value_increase, "\t", b.value_decrease, "\t", b.value_increase_or_decrease, "\t", b.value_inflow, "\t",
				b.value_outflow, "\t", b.value_flow, "\t", b.value_turnover, "\t", b.value_percent, "\t", b.value_change_since_base_period, "\t", b.value_growth_ratio_to_base_period, "\t",
				b.price_ending_balance, "\t", b.price_average, "\t", b.price_increase, "\t", b.price_decrease, "\t", b.price_increase_or_decrease, "\t", b.price_inflow, "\t",
				b.price_outflow, "\t", b.price_flow, "\t", b.price_turnover, "\t", b.price_percent, "\t", b.price_change_since_base_period, "\t", b.price_growth_ratio_to_base_period, "\t",
				b.quantity_ending_balance, "\t", b.quantity_average, "\t", b.quantity_increase, "\t", b.quantity_decrease, "\t", b.quantity_increase_or_decrease, "\t", b.quantity_inflow, "\t",
				b.quantity_outflow, "\t", b.quantity_flow, "\t", b.quantity_turnover, "\t", b.quantity_percent, "\t", b.quantity_change_since_base_period, "\t", b.quantity_growth_ratio_to_base_period, "\t")
		}
		fmt.Fprintln(w, "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t", "\t")
	}
	// w.Flush()

	// t := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// fmt.Fprintln(t, "date\t", "entry_number\t", "account\t", "value\t", "price\t", "quantity\t", "barcode\t", "entry_expair\t", "description\t", "name\t", "employee_name\t", "entry_date\t", "reverse")
	// for _, a := range journal_tag {
	// 	fmt.Fprintln(t, a.date, "\t", a.entry_number, "\t", a.account, "\t", a.value, "\t", a.price, "\t", a.quantity, "\t", a.barcode, "\t", a.entry_expair, "\t", a.description, "\t", a.name, "\t", a.employee_name, "\t", a.entry_date, "\t", a.reverse, "\t")
	// }
	// t.Flush()

	p := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(p, "current_ratio\t", "acid_test\t", "receivables_turnover\t", "inventory_turnover\t", "profit_margin\t", "asset_turnover\t", "return_on_assets\t", "return_on_equity\t", "return_on_common_stockholders_equity\t", "earnings_per_share\t", "price_earnings_ratio\t", "payout_ratio\t", "debt_to_total_assets_ratio\t", "times_interest_earned\t")
	for _, a := range financial_analysis_statement {
		fmt.Fprintln(p, a.current_ratio, "\t", a.acid_test, "\t", a.receivables_turnover, "\t", a.inventory_turnover, "\t", a.profit_margin, "\t", a.asset_turnover, "\t", a.return_on_assets, "\t", a.return_on_equity, "\t", a.return_on_common_stockholders_equity, "\t", a.earnings_per_share, "\t", a.price_earnings_ratio, "\t", a.payout_ratio, "\t", a.debt_to_total_assets_ratio, "\t", a.times_interest_earned, "\t")
	}
	// p.Flush()

	r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	for _, v := range all_flows_for_all {
		fmt.Fprintln(r, "/////////////////////////////////////////////////////////////////////////////////////////////")
		for keya, a := range v {
			for keyb, b := range a {
				for keyc, c := range b {
					for keyd, d := range c {
						for keye, e := range d {
							if keyb == "inventory" && keyd == "value" && keye == "turnover_days" {
								fmt.Fprintln(r, keya, "\t", keyb, "\t", keyc, "\t", keyd, "\t", keye, "\t", e)
							}
						}
					}
				}
			}
		}
	}
	r.Flush()

	a1, ok1 := all_flows_for_all[0]["cash"]["assets"]["yasa"]["value"]["ending_balance"]
	// a2, ok2 := all_flows_for_all[0]["cash"]["cash"]["yasa"]["value"]["ending_balance"]
	// a3, ok3 := all_flows_for_all[0]["cash"]["tax"]["yasa"]["value"]["ending_balance"]
	fmt.Println(a1, ok1)
	// fmt.Println(a2, ok2)
	// fmt.Println(a3, ok3)

	// point := Managerial_Accounting{
	// 	cvp:      []cvp{{"falafel", 1000, 0, 1250, 500, 0, []float64{0}}},
	// 	overhead: []overhead{{"fixed", "units", []float64{300000 / 12, 8000 * 10, 100000}}},
	// }
	// j := point.cost_volume_profit_slice()
	// q := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// fmt.Fprintln(q, "name\t", "units\t", "selling_price_per_unit\t", "variable_cost_per_unit\t", "fixed_cost\t", "mixed_cost\t", "mixed_cost_per_unit\t", "sales\t", "profit\t", "profit_per_unit\t", "contribution_margin_per_unit\t", "contribution_margin\t", "contribution_margin_ratio\t", "break_even_in_unit\t", "break_even_in_sales\t", "degree_of_operating_leverage\t")
	// for _, i := range j {
	// 	fmt.Fprintln(q, i.name, "\t", i.units, "\t", i.selling_price_per_unit, "\t", i.variable_cost_per_unit, "\t", i.fixed_cost, "\t", i.mixed_cost, "\t", i.mixed_cost_per_unit, "\t", i.sales, "\t", i.profit, "\t", i.profit_per_unit, "\t", i.contribution_margin_per_unit, "\t", i.contribution_margin, "\t", i.contribution_margin_ratio, "\t", i.break_even_in_unit, "\t", i.break_even_in_sales, "\t", i.degree_of_operating_leverage, "\t")
	// }
	// q.Flush()
}
