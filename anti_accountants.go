package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"sort"
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

type Financial_accounting struct {
	DriverName, DataSourceName, Database_name           string
	Invoice_discounts_list                              [][2]float64
	retained_earnings, income_summary, invoice_discount string
	Assets_normal, Current_assets, Cash_and_cash_equivalent, Fifo, Lifo, Wma, Short_term_investments, Receivables, Assets_contra, Allowance_for_Doubtful_Accounts, Liabilities_normal, Current_liabilities, Liabilities_contra,
	Equity_normal, Equity_contra, Withdrawals, Sales, Revenues, Discounts, Sales_returns_and_allowances, Expenses []string
	auto_complete_entries [][]account_method_value_price
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
	value, price, quantity, percent, average, turnover,
	value_base, price_base, quantity_base, percent_base, average_base, turnover_base,
	changes_since_base_period, current_period_in_relation_to_base_period float64
}

type financial_analysis struct {
	current_assets, current_liabilities, cash, short_term_investments, net_receivables, net_credit_sales,
	average_net_receivables, cost_of_goods_sold, average_inventory, net_income, net_sales, average_assets,
	preferred_dividends, average_common_stockholders_equity, weighted_average_common_shares_outstanding,
	market_price_per_shares_outstanding, cash_dividends, total_debt, total_assets,
	income_before_income_taxes_and_interest_expense, interest_expense float64
}

type financial_analysis_statement struct {
	ratio                     string
	current_value, value_base float64
	formula, purpose_or_use   string
}

type value_quantity struct {
	value, quantity float64
}

type cvp_statistics struct {
	units,
	selling_price_per_unit,
	variable_cost_per_unit,
	fixed_cost,
	mixed_cost,
	mixed_cost_per_unit,
	sales,
	profit,
	profit_per_unit,
	contribution_margin,
	contribution_margin_ratio,
	break_even_in_unit,
	break_even_in_sales,
	degree_of_operating_leverage float64
}

type cvp struct {
	units, selling_price_per_unit, variable_cost_per_unit, fixed_cost float64
}

type Managerial_Accounting struct {
	points_activity_level_and_cost_at_the_activity_level [][2]float64
	cvp                                                  []cvp
	fixed_cost,
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
	inventory, current_assets, assets_normal, assets_contra, liabilities_normal, equity_normal, revenues, expenses,
	temporary_debit_accounts, temporary_accounts, debit_accounts, credit_accounts []string
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
	db.Exec("create table if not exists inventory (date text,entry_number integer,account text,price real,quantity real,quantity_remaining real,adjusted_cost real,barcode text,entry_expair text,name text,employee_name text,entry_date text)")

	inventory = concat(s.Fifo, s.Lifo, s.Wma).([]string)
	current_assets = concat(s.Current_assets, inventory, s.Cash_and_cash_equivalent, s.Short_term_investments, s.Receivables).([]string)
	assets_normal = concat(s.Assets_normal, current_assets).([]string)
	assets_contra = concat(s.Assets_contra, s.Allowance_for_Doubtful_Accounts).([]string)
	liabilities_normal = concat(s.Liabilities_normal, s.Current_liabilities).([]string)
	equity_normal = append(s.Equity_normal, s.retained_earnings, s.income_summary)
	revenues = concat(s.Revenues, s.Sales).([]string)
	expenses = concat(s.Expenses, s.Sales_returns_and_allowances, s.Discounts).([]string)
	temporary_debit_accounts = concat(s.Withdrawals, expenses).([]string)
	temporary_accounts = concat(temporary_debit_accounts, revenues).([]string)
	debit_accounts = concat(assets_normal, s.Liabilities_contra, s.Equity_contra, temporary_debit_accounts).([]string)
	credit_accounts = concat(assets_contra, liabilities_normal, equity_normal, revenues).([]string)
	all_accounts := concat(debit_accounts, credit_accounts).([]string)

	entry_number := entry_number()
	var array_to_insert []journal_tag
	expair_expenses := journal_tag{Now.String(), entry_number, "expair_expenses", 0, 0, 0, "", time.Time{}.String(), "to record the expiry of the goods automatically", "", "", Now.String(), false}
	expair_goods, _ := db.Query("select account,price*quantity*-1,price,quantity*-1,barcode from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", Now.String())
	for expair_goods.Next() {
		tag := expair_expenses
		expair_goods.Scan(&tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode)
		expair_expenses.value -= tag.value
		expair_expenses.quantity -= tag.quantity
		array_to_insert = append(array_to_insert, tag)
	}
	expair_expenses.price = expair_expenses.value / expair_expenses.quantity
	array_to_insert = append(array_to_insert, expair_expenses)
	s.insert_to_database(array_to_insert, true, false, false)
	db.Exec("delete from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", Now.String())
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

	check_accounts("account", "inventory", " is not on fifo lifo wma parameters accounts lists", inventory)
	check_if_duplicates(all_accounts)
}

func (s Financial_accounting) journal_entry(array_of_entry []Account_value_quantity_barcode, auto_completion bool, date time.Time, entry_expair time.Time, adjusting_method string,
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
			} else if is_in(entry.Account, s.Discounts) {
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
	return array_to_insert
}

func (s Financial_accounting) financial_statements(end_base_date, start_date, end_date time.Time) ([]statement, []statement, []statement, []financial_analysis_statement) {
	check_dates(start_date, end_date)
	check_dates(end_base_date, start_date)
	d1 := int(end_date.Sub(start_date).Hours() / 24)
	start_base_date := end_base_date.AddDate(0, 0, -d1)
	d2 := int(end_base_date.Sub(start_base_date).Hours() / 24)
	var journal []journal_tag
	rows, _ := db.Query("select date,entry_number,account,value,quantity from journal order by date")
	for rows.Next() {
		var entry journal_tag
		rows.Scan(&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.quantity)
		journal = append(journal, entry)
	}
	journal_map1, income_map1, cash_map1, v_net_credit_sales1 := s.prepare_statment_map(journal, start_date, end_date)
	journal_map2, income_map2, cash_map2, _ := s.prepare_statment_map(journal, start_date.AddDate(0, 0, -d1), end_date.AddDate(0, 0, -d1))
	journal_map3, income_map3, cash_map3, v_net_credit_sales3 := s.prepare_statment_map(journal, start_base_date, end_base_date)
	journal_map4, income_map4, cash_map4, _ := s.prepare_statment_map(journal, start_base_date.AddDate(0, 0, -d2), end_base_date.AddDate(0, 0, -d2))
	c, cash_increase := s.prepare_analysis(journal_map1, income_map1, cash_map1, v_net_credit_sales1)
	b, cash_increase_base := s.prepare_analysis(journal_map3, income_map3, cash_map3, v_net_credit_sales3)
	analysis := []financial_analysis_statement{
		{"current_ratio", c.current_ratio(), b.current_ratio(), "current_assets / current_liabilities", "Measures short-term debt-paying ability"},
		{"acid_test", c.acid_test(), b.acid_test(), "(cash + short_term_investments + net_receivables) / current_liabilities", "Measures immediate short term liquidity"},
		{"receivables_turnover", c.receivables_turnover(), b.receivables_turnover(), "net_credit_sales / average_net_receivables", "Measures liquidity of receivables"},
		{"inventory_turnover", c.inventory_turnover(), b.inventory_turnover(), "cost_of_goods_sold / average_inventory", "Measures liquidity of inventory"},
		{"profit_margin", c.profit_margin(), b.profit_margin(), "net_income / net_sales", "Measures net income generated by each dollar of sales"},
		{"asset_turnover", c.asset_turnover(), b.asset_turnover(), "net_sales / average_assets", "Measures how efficiently assets are used to generate sales"},
		{"return_on_assets", c.return_on_assets(), b.return_on_assets(), "net_income / average_assets", "Measures overall profitability of assets"},
		{"return_on_common_stockholders_equity", c.return_on_common_stockholders_equity(), b.return_on_common_stockholders_equity(), "(net_income - preferred_dividends) / average_common_stockholders_equity", "Measures profitability of owners investment"},
		{"earnings_per_share", c.earnings_per_share(), b.earnings_per_share(), "(net_income - preferred_dividends) / weighted_average_common_shares_outstanding", "Measures net income earned on each share of common stock"},
		{"price_earnings_ratio", c.price_earnings_ratio(), b.price_earnings_ratio(), "market_price_per_shares_outstanding / earnings_per_share", "Measures the ratio of the market price per share to earnings per share"},
		{"payout_ratio", c.payout_ratio(), b.payout_ratio(), "cash_dividends / net_income", "Measures percentage of earnings distributed in the form of cash dividends"},
		{"debt_to_total_assets_ratio", c.debt_to_total_assets_ratio(), b.debt_to_total_assets_ratio(), "total_debt / total_assets", "Measures the percentage of total assets provided by creditors"},
		{"times_interest_earned", c.times_interest_earned(), b.times_interest_earned(), "income_before_income_taxes_and_interest_expense / interest_expense", "Measures ability to meet interest payments as they come due"},
	}
	cash_flow := remove_empties_lines(prepare_statement(cash_map1, cash_map2, cash_map3, cash_map4, cash_increase, cash_increase_base))
	balance_sheet := remove_empties_lines(prepare_statement(journal_map1, journal_map2, journal_map3, journal_map4, c.total_assets, b.total_assets))
	income_statements := remove_empties_lines(prepare_statement(income_map1, income_map2, income_map3, income_map4, c.net_sales, b.net_sales))
	sort.Slice(cash_flow[:], func(i, j int) bool { return cash_flow[i].value > cash_flow[j].value })
	sort.Slice(balance_sheet[:], func(i, j int) bool { return balance_sheet[i].value > balance_sheet[j].value })
	sort.Slice(income_statements[:], func(i, j int) bool { return income_statements[i].value > income_statements[j].value })

	return balance_sheet, income_statements, cash_flow, analysis
}

func (s Financial_accounting) invoice(array_of_journal_tag []journal_tag) []invoice_struct {
	m := map[string]*invoice_struct{}
	for _, entry := range array_of_journal_tag {
		var key string
		switch {
		case is_in(entry.account, assets_normal) && !is_in(entry.account, inventory) && entry.value > 0:
			key = "total"
		case is_in(entry.account, s.Discounts):
			key = "total discounts"
		case is_in(entry.account, s.Sales):
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
			if parse_date(entry.date).Before(Now) {
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
	case is_in(account, s.Fifo):
		order_by_date_asc_or_desc = "asc"
	case is_in(account, s.Lifo):
		order_by_date_asc_or_desc = "desc"
	case is_in(account, s.Wma):
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

func (s Financial_accounting) prepare_statment_map(journal []journal_tag, start_date, end_date time.Time) (map[string]*value_quantity, map[string]*value_quantity, map[string]*value_quantity, float64) {
	var cash []journal_tag
	var ok, is_receivables bool
	var previous_date string
	var v_net_credit_sales, net_credit_sales_box float64
	journal_map := map[string]*value_quantity{s.retained_earnings: {0, 0}, s.income_summary: {0, 0}}
	income_map := map[string]*value_quantity{}
	cash_map := map[string]*value_quantity{}
	for _, entry := range journal {
		key_journal := entry.account
		sum_journal := journal_map[key_journal]
		if sum_journal == nil {
			sum_journal = &value_quantity{}
			journal_map[key_journal] = sum_journal
		}
		key_income := entry.account
		sum_income := income_map[key_income]
		if sum_income == nil {
			sum_income = &value_quantity{}
			income_map[key_income] = sum_income
		}
		date := parse_date(entry.date)
		if previous_date != entry.date {
			if ok {
				for _, entry := range cash {
					key_cash := entry.account
					sum_cash := cash_map[key_cash]
					if sum_cash == nil {
						sum_cash = &value_quantity{}
						cash_map[key_cash] = sum_cash
					}
					if is_in(entry.account, credit_accounts) {
						sum_cash.value += entry.value
						sum_cash.quantity += entry.quantity
					} else {
						sum_cash.value -= entry.value
						sum_cash.quantity -= entry.quantity
					}
				}
			}
			cash = []journal_tag{}
			ok = false
			if is_receivables {
				v_net_credit_sales += net_credit_sales_box
			}
			net_credit_sales_box = 0
			is_receivables = false
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
				sum_journal.quantity += entry.quantity
			default:
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
			}
		}
		if date.After(start_date) && date.Before(end_date) {
			if is_in(entry.account, s.Cash_and_cash_equivalent) {
				ok = true
			} else {
				cash = append(cash, entry)
			}
			if is_in(entry.account, s.Receivables) {
				is_receivables = true
			}
			switch {
			case is_in(entry.account, revenues):
				sum_journal := journal_map[s.income_summary]
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
				sum_income.value += entry.value
				sum_income.quantity += entry.quantity
				if is_in(entry.account, s.Sales) {
					net_credit_sales_box += entry.value
				}
			case is_in(entry.account, expenses):
				sum_journal := journal_map[s.income_summary]
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
				sum_income.value += entry.value
				sum_income.quantity += entry.quantity
				if is_in(entry.account, s.Sales_returns_and_allowances) {
					net_credit_sales_box -= entry.value
				}
			default:
				sum_journal.value += entry.value
				sum_journal.quantity += entry.quantity
			}
		}
	}
	return journal_map, income_map, cash_map, v_net_credit_sales
}

func (s Financial_accounting) prepare_analysis(journal_map, income_map, cash_map map[string]*value_quantity, v_net_credit_sales float64) (financial_analysis, float64) {
	var v_current_assets, v_current_liabilities, v_cash, v_short_term_investments, v_net_receivables, v_average_net_receivables, v_cost_of_goods_sold, v_average_inventory, v_net_income, v_net_sales,
		v_average_assets, v_preferred_dividends, v_average_common_stockholders_equity, v_weighted_average_common_shares_outstanding, v_market_price_per_shares_outstanding, v_cash_dividends, v_total_debt, v_total_assets,
		v_income_before_income_taxes_and_interest_expense, v_interest_expense float64
	var cash_increase float64
	for key, v := range journal_map {
		switch {
		case is_in(key, assets_normal):
			v_total_assets += v.value
			if is_in(key, current_assets) {
				v_current_assets += v.value
				if is_in(key, s.Short_term_investments) {
					v_short_term_investments += v.value
				}
				if is_in(key, s.Receivables) {
					v_net_receivables += v.value
				}
				if is_in(key, s.Cash_and_cash_equivalent) {
					v_cash += v.value
				}
			}
		case is_in(key, assets_contra):
			v_total_assets -= v.value
			if is_in(key, s.Allowance_for_Doubtful_Accounts) {
				v_net_receivables -= v.value
			}
		case is_in(key, liabilities_normal):
			v_total_debt += v.value
			if is_in(key, s.Current_liabilities) {
				v_current_liabilities += v.value
			}
		case is_in(key, s.Liabilities_contra):
			v_total_debt -= v.value
		}
	}
	for key, v := range income_map {
		if is_in(key, revenues) {
			v_net_income += v.value
			if is_in(key, s.Sales) {
				v_net_sales += v.value
			}
		} else {
			v_net_income -= v.value
			if is_in(key, s.Sales_returns_and_allowances) || is_in(key, s.Discounts) {
				v_net_sales -= v.value
			}
		}
	}
	for key, v := range cash_map {
		if is_in(key, credit_accounts) {
			cash_increase += v.value
		}
	}
	a := financial_analysis{
		current_assets:                     v_current_assets,
		current_liabilities:                v_current_liabilities,
		cash:                               v_cash,
		short_term_investments:             v_short_term_investments,
		net_receivables:                    v_net_receivables,
		net_credit_sales:                   v_net_credit_sales,
		average_net_receivables:            v_average_net_receivables,
		cost_of_goods_sold:                 v_cost_of_goods_sold,
		average_inventory:                  v_average_inventory,
		net_income:                         v_net_income,
		net_sales:                          v_net_sales,
		average_assets:                     v_average_assets,
		preferred_dividends:                v_preferred_dividends,
		average_common_stockholders_equity: v_average_common_stockholders_equity,
		weighted_average_common_shares_outstanding:      v_weighted_average_common_shares_outstanding,
		market_price_per_shares_outstanding:             v_market_price_per_shares_outstanding,
		cash_dividends:                                  v_cash_dividends,
		total_debt:                                      v_total_debt,
		total_assets:                                    v_total_assets,
		income_before_income_taxes_and_interest_expense: v_income_before_income_taxes_and_interest_expense,
		interest_expense:                                v_interest_expense,
	}
	return a, cash_increase
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

func prepare_statement(statement_map1, statement_map2, statement_map3, statement_map4 map[string]*value_quantity, amount, amount_base float64) []statement {
	statement_map := map[string]*statement{}
	for key, v := range statement_map1 {
		key_statement := key
		sum_statement := statement_map[key_statement]
		if sum_statement == nil {
			sum_statement = &statement{}
			statement_map[key_statement] = sum_statement
		}
		sum_statement.value += v.value
		sum_statement.quantity += v.quantity
		sum_statement.average += v.value / 2
	}
	for key, v := range statement_map2 {
		key_statement := key
		sum_statement := statement_map[key_statement]
		if sum_statement == nil {
			sum_statement = &statement{}
			statement_map[key_statement] = sum_statement
		}
		sum_statement.average += v.value / 2
	}
	for key, v := range statement_map3 {
		key_statement := key
		sum_statement := statement_map[key_statement]
		if sum_statement == nil {
			sum_statement = &statement{}
			statement_map[key_statement] = sum_statement
		}
		sum_statement.value_base += v.value
		sum_statement.quantity_base += v.quantity
		sum_statement.average_base += v.value / 2
	}
	for key, v := range statement_map4 {
		key_statement := key
		sum_statement := statement_map[key_statement]
		if sum_statement == nil {
			sum_statement = &statement{}
			statement_map[key_statement] = sum_statement
		}
		sum_statement.average_base += v.value / 2
	}

	var statement_sheet []statement
	for key, v := range statement_map {
		statement_sheet = append(statement_sheet, statement{
			account:                   key,
			value:                     v.value,
			price:                     v.value / v.quantity,
			quantity:                  v.quantity,
			percent:                   v.value / amount,
			average:                   v.average,
			turnover:                  0,
			value_base:                v.value_base,
			price_base:                v.value_base / v.quantity_base,
			quantity_base:             v.quantity_base,
			percent_base:              v.value_base / amount_base,
			average_base:              v.average_base,
			turnover_base:             0,
			changes_since_base_period: v.value - v.value_base,
			current_period_in_relation_to_base_period: v.value / v.value_base,
		})
	}
	return statement_sheet
}

func remove_empties_lines(statement []statement) []statement {
	var index int
	for index < len(statement) {
		if (statement[index].value == 0 || math.IsNaN(statement[index].value)) &&
			(statement[index].price == 0 || math.IsNaN(statement[index].price)) &&
			(statement[index].quantity == 0 || math.IsNaN(statement[index].quantity)) &&
			(statement[index].percent == 0 || math.IsNaN(statement[index].percent)) &&
			(statement[index].average == 0 || math.IsNaN(statement[index].average)) &&
			(statement[index].value_base == 0 || math.IsNaN(statement[index].value_base)) &&
			(statement[index].price_base == 0 || math.IsNaN(statement[index].price_base)) &&
			(statement[index].quantity_base == 0 || math.IsNaN(statement[index].quantity_base)) &&
			(statement[index].percent_base == 0 || math.IsNaN(statement[index].percent_base)) &&
			(statement[index].average_base == 0 || math.IsNaN(statement[index].average_base)) &&
			(statement[index].changes_since_base_period == 0 || math.IsNaN(statement[index].changes_since_base_period)) &&
			(statement[index].current_period_in_relation_to_base_period == 0 || math.IsNaN(statement[index].current_period_in_relation_to_base_period)) {
			statement = append(statement[:index], statement[index+1:]...)
		} else {
			index++
		}
	}
	return statement
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

func parse_date(string_date string) time.Time {
	date, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 +03 m=+0.999999999", string_date)
	if err != nil {
		date, _ = time.Parse("2006-01-02 15:04:05.999999999 -0700 +03", string_date)
	}
	return date
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
	for _, entry := range array_of_entry {
		switch {
		case is_in(entry.Account, debit_accounts):
			zero += entry.value
		case is_in(entry.Account, credit_accounts):
			zero -= entry.value
		default:
			log.Panic(entry.Account, " is not on parameters accounts lists")
		}
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

// current_assets / current_liabilities Measures short-term debt-paying ability
func (s financial_analysis) current_ratio() float64 {
	return s.current_assets / s.current_liabilities
}

// (cash + short_term_investments + net_receivables) / current_liabilities Measures immediate short term liquidity
func (s financial_analysis) acid_test() float64 {
	return (s.cash + s.short_term_investments + s.net_receivables) / s.current_liabilities
}

// net_credit_sales / average_net_receivables Measures liquidity of receivables
func (s financial_analysis) receivables_turnover() float64 {
	return s.net_credit_sales / s.average_net_receivables
}

// cost_of_goods_sold / average_inventory Measures liquidity of inventory
func (s financial_analysis) inventory_turnover() float64 {
	return s.cost_of_goods_sold / s.average_inventory
}

// net_income / net_sales Measures net income generated by each dollar of sales
func (s financial_analysis) profit_margin() float64 {
	return s.net_income / s.net_sales
}

// net_sales / average_assets Measures how efficiently assets are used to generate sales
func (s financial_analysis) asset_turnover() float64 {
	return s.net_sales / s.average_assets
}

// net_income / average_assets Measures overall profitability of assets
func (s financial_analysis) return_on_assets() float64 {
	return s.net_income / s.average_assets
}

// (net_income - preferred_dividends) / average_common_stockholders_equity Measures profitability of owners investment
func (s financial_analysis) return_on_common_stockholders_equity() float64 {
	return (s.net_income - s.preferred_dividends) / s.average_common_stockholders_equity
}

// (net_income - preferred_dividends) / weighted_average_common_shares_outstanding Measures net income earned on each share of common stock
func (s financial_analysis) earnings_per_share() float64 {
	return (s.net_income - s.preferred_dividends) / s.weighted_average_common_shares_outstanding
}

// market_price_per_shares_outstanding / earnings_per_share Measures the ratio of the market price per share to earnings per share
func (s financial_analysis) price_earnings_ratio() float64 {
	return s.market_price_per_shares_outstanding / s.earnings_per_share()
}

// cash_dividends / net_income Measures percentage of earnings distributed in the form of cash dividends
func (s financial_analysis) payout_ratio() float64 {
	return s.cash_dividends / s.net_income
}

// total_debt / total_assets Measures the percentage of total assets provided by creditors
func (s financial_analysis) debt_to_total_assets_ratio() float64 {
	return s.total_debt / s.total_assets
}

// income_before_income_taxes_and_interest_expense / interest_expense Measures ability to meet interest payments as they come due
func (s financial_analysis) times_interest_earned() float64 {
	return s.income_before_income_taxes_and_interest_expense / s.interest_expense
}

func (s Managerial_Accounting) decrease() float64 {
	return s.beginning_balance + s.increase - s.ending_balance
}

func (s Managerial_Accounting) cost_of_goods_sold() float64 {
	return s.decrease() - s.decreases_in_account_caused_by_not_sell
}

func (s Managerial_Accounting) c_v_p() ([]cvp_statistics, float64, float64, float64, float64) {
	var j []cvp_statistics
	var total_sales, total_contribution_margin, total_contribution_margin_ratio, total_break_even_in_sales float64
	for _, i := range s.cvp {
		mixed_cost := i.fixed_cost + i.variable_cost_per_unit*i.units
		mixed_cost_per_unit := mixed_cost / i.units
		sales := i.selling_price_per_unit * i.units
		profit := i.selling_price_per_unit*i.units - mixed_cost
		profit_per_unit := i.selling_price_per_unit - i.variable_cost_per_unit
		contribution_margin := profit_per_unit * i.units
		contribution_margin_ratio := profit_per_unit / i.selling_price_per_unit
		break_even_in_unit := i.fixed_cost / profit_per_unit
		break_even_in_sales := break_even_in_unit * i.selling_price_per_unit
		degree_of_operating_leverage := contribution_margin / profit
		j = append(j, cvp_statistics{
			units:                        i.units,
			selling_price_per_unit:       i.selling_price_per_unit,
			variable_cost_per_unit:       i.variable_cost_per_unit,
			fixed_cost:                   i.fixed_cost,
			mixed_cost:                   mixed_cost,
			mixed_cost_per_unit:          mixed_cost_per_unit,
			sales:                        sales,
			profit:                       profit,
			profit_per_unit:              profit_per_unit,
			contribution_margin:          contribution_margin,
			contribution_margin_ratio:    contribution_margin_ratio,
			break_even_in_unit:           break_even_in_unit,
			break_even_in_sales:          break_even_in_sales,
			degree_of_operating_leverage: degree_of_operating_leverage,
		})
		total_sales += sales
		total_contribution_margin += contribution_margin
	}
	total_contribution_margin_ratio = total_contribution_margin / total_sales
	total_break_even_in_sales = s.fixed_cost / total_contribution_margin_ratio
	for index, i := range j {
		j[index].break_even_in_sales = (i.sales / total_sales) * (total_break_even_in_sales + i.break_even_in_sales)
		j[index].break_even_in_unit = j[index].break_even_in_sales / i.selling_price_per_unit
	}
	return j, total_sales, total_contribution_margin, total_contribution_margin_ratio, total_break_even_in_sales
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

// the first column is for the quantity and the second and third represent price, if the quantity represent the revenue keep it positive else if it represent the cost make the quantity negative
func differential_cost_and_revenue(quantity_present_price_proposed_price [][3]float64) float64 {
	var present_income, proposed_income float64
	for _, i := range quantity_present_price_proposed_price {
		present_income += i[0] * i[1]
		proposed_income += i[0] * i[2]
	}
	return proposed_income - present_income
}

func main() {
	_ = Financial_accounting{
		DriverName:                      "mysql",
		DataSourceName:                  "hashem:hashem@tcp(localhost)/",
		Database_name:                   "acc",
		Invoice_discounts_list:          [][2]float64{{5, -10}},
		retained_earnings:               "retained_earnings",
		income_summary:                  "income_summary",
		invoice_discount:                "invoice_discount",
		Assets_normal:                   []string{},
		Current_assets:                  []string{},
		Cash_and_cash_equivalent:        []string{"cash"},
		Fifo:                            []string{"book"},
		Lifo:                            []string{},
		Wma:                             []string{},
		Short_term_investments:          []string{},
		Receivables:                     []string{},
		Assets_contra:                   []string{},
		Allowance_for_Doubtful_Accounts: []string{},
		Liabilities_normal:              []string{},
		Current_liabilities:             []string{"tax"},
		Liabilities_contra:              []string{},
		Equity_normal:                   []string{},
		Equity_contra:                   []string{},
		Withdrawals:                     []string{},
		Sales:                           []string{"service revenue", "revenue of book"},
		Revenues:                        []string{},
		Discounts:                       []string{"discount of book", "invoice_discount", "service_discount"},
		Sales_returns_and_allowances:    []string{},
		Expenses:                        []string{"cost of book", "tax of book", "tax of service revenue", "invoice_tax"},
		auto_complete_entries: [][]account_method_value_price{{{"service revenue", "quantity_ratio", 0, 10}, {"tax of service revenue", "value", 1, 1}, {"tax", "value", 1, 1}, {"service_discount", "value", 1, 1}},
			{{"book", "quantity_ratio", -1, 0}, {"revenue of book", "quantity_ratio", 1, 10}, {"cost of book", "copy_abs", 0, 0}, {"tax of book", "value", 1, 1}, {"tax", "value", 1, 1}, {"discount of book", "value", 1, 1}}},
	}
	// v.initialize()
	// entry := v.journal_entry([]Account_value_quantity_barcode{{"service revenue", 10, 100, ""}, {"cash", 989, 989, ""}}, true, Now,
	// 	time.Time{}, "", "", "yasa", "hashem", []day_start_end{})
	// v.insert_to_database(entry, true, true, true)

	// entry := select_journal(0, "cash", time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local))
	// fmt.Println(v.invoice(entry))
	// reverse_entry(2, time.Time{}, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local), "hashem")
	// r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// for _, i := range entry {
	// 	fmt.Fprintln(r, "\t", i.date, "\t", i.entry_number, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.barcode, "\t", i.entry_expair, "\t", i.description, "\t", i.name, "\t", i.employee_name, "\t", i.entry_date, "\t", i.reverse)
	// }
	// r.Flush()

	// balance_sheet, income_statements, cash_flow, analysis := v.financial_statements(
	// 	time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local),
	// 	time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local),
	// 	time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local))

	// w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// for _, i := range balance_sheet {
	// 	fmt.Fprintln(w, "balance_sheet\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.average, "\t", i.turnover, "\t", i.value_base, "\t", i.price_base, "\t", i.quantity_base, "\t", i.percent_base, "\t", i.average_base, "\t", i.turnover_base, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	// }
	// for _, i := range income_statements {
	// 	fmt.Fprintln(w, "income_statements\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.average, "\t", i.turnover, "\t", i.value_base, "\t", i.price_base, "\t", i.quantity_base, "\t", i.percent_base, "\t", i.average_base, "\t", i.turnover_base, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	// }
	// for _, i := range cash_flow {
	// 	fmt.Fprintln(w, "cash_flow\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.average, "\t", i.turnover, "\t", i.value_base, "\t", i.price_base, "\t", i.quantity_base, "\t", i.percent_base, "\t", i.average_base, "\t", i.turnover_base, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	// }
	// fmt.Fprintln(w, "######################################################################### analysis ##########################################################################")
	// for _, i := range analysis {
	// 	fmt.Fprintln(w, i.ratio, "\t", i.current_value, "\t", i.value_base, "\t", i.formula, "\t", i.purpose_or_use)
	// }
	// w.Flush()

	point := Managerial_Accounting{
		points_activity_level_and_cost_at_the_activity_level: [][2]float64{{2310, 10113}, {2453, 12691}, {2641, 10905}, {2874, 12949}, {3540, 15334}, {4861, 21455}, {5432, 21270}, {5268, 19930}, {4628, 21860}, {3720, 18383}, {2106, 9830}, {2495, 11081}},
		cvp:                                     []cvp{{1, 20000, 15000, 500}, {1, 80000, 40000, 0}},
		fixed_cost:                              27000,
		beginning_balance:                       200,
		increase:                                5000,
		ending_balance:                          400,
		decreases_in_account_caused_by_not_sell: 30,
		actual_mixed_cost:                       0,
		number_of_partially_completed_units:     0,
		percentage_completion:                   0,
		units_transferred_to_the_next_department_or_to_finished_goods: 4800,
		equivalent_units_in_ending_work_in_process_inventory:          160,
		equivalent_units_in_beginning_work_in_process_inventory:       0,
		cost_of_beginning_work_in_process_inventory:                   0,
		cost_added_during_the_period:                                  368600,
	}
	// fmt.Println(differential_cost_and_revenue([][3]float64{
	// 	{1, 700000, 800000},
	// 	{-1, 350000, 400000},
	// 	{-1, 80000, 45000},
	// 	{-1, 0, 40000},
	// 	{-1, 50000, 80000},
	// 	{-1, 60000, 60000},
	// }))
	fmt.Println("decrease", point.decrease())
	fmt.Println("cost_of_goods_sold", point.cost_of_goods_sold())
	fmt.Println("equivalent_units", point.equivalent_units())
	fmt.Println("equivalent_units_of_production_weighted_average_method", point.equivalent_units_of_production_weighted_average_method())
	fmt.Println("cost_per_equivalent_unit_weighted_average_method", point.cost_per_equivalent_unit_weighted_average_method())
	fmt.Println("equivalent_units_of_production_fifo_method", point.equivalent_units_of_production_fifo_method())
	fmt.Println("equivalent_units_to_complete_beginning_work_in_process_inventory", point.equivalent_units_to_complete_beginning_work_in_process_inventory())
	fmt.Println("cost_per_equivalent_unit_fifo_method", point.cost_per_equivalent_unit_fifo_method())
	fmt.Println("high_low", point.high_low())
	fmt.Println(point.least_squares_regression())

	j, total_sales, total_contribution_margin, total_contribution_margin_ratio, total_break_even_in_sales := point.c_v_p()
	fmt.Println("total_sales", total_sales)
	fmt.Println("total_contribution_margin", total_contribution_margin)
	fmt.Println("total_contribution_margin_ratio", total_contribution_margin_ratio)
	fmt.Println("total_break_even_in_sales", total_break_even_in_sales)
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, "units\t", "selling_price_per_unit\t", "variable_cost_per_unit\t", "fixed_cost\t", "mixed_cost\t", "mixed_cost_per_unit\t", "sales\t", "profit\t", "profit_per_unit\t", "contribution_margin\t", "contribution_margin_ratio\t", "break_even_in_unit\t", "break_even_in_sales\t", "degree_of_operating_leverage\t")
	for _, i := range j {
		fmt.Fprintln(w, i.units, "\t", i.selling_price_per_unit, "\t", i.variable_cost_per_unit, "\t", i.fixed_cost, "\t", i.mixed_cost, "\t", i.mixed_cost_per_unit, "\t", i.sales, "\t", i.profit, "\t", i.profit_per_unit, "\t", i.contribution_margin, "\t", i.contribution_margin_ratio, "\t", i.break_even_in_unit, "\t", i.break_even_in_sales, "\t", i.degree_of_operating_leverage, "\t")
	}
	w.Flush()
}
