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

type filtered_statement struct {
	key_account_flow, key_account, key_name, key_vpq, key_number string
	number                                                       float64
}

type one_step_distribution struct {
	sales_or_variable_or_fixed, distribution_method string
	from, to                                        map[string]float64
}

type Managerial_Accounting struct {
	points_activity_level_and_cost_at_the_activity_level [][2]float64
	cvp                                                  map[string]map[string]float64
	distribution_steps                                   []one_step_distribution
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
	rows, _ := db.Query("select entry_number,account,value from journal order by date,entry_number")
	for rows.Next() {
		var entry_number int
		var tag Account_value_quantity_barcode
		rows.Scan(&entry_number, &tag.Account, &tag.value)
		if previous_entry_number != entry_number {
			s.check_debit_equal_credit(double_entry, true)
			journal = append(journal, double_entry)
			double_entry = []Account_value_quantity_barcode{}
		}
		double_entry = append(double_entry, tag)
		previous_entry_number = entry_number
	}
}

func (s Financial_accounting) journal_entry(array_of_entry []Account_value_quantity_barcode, insert, auto_completion bool, date time.Time, entry_expair time.Time, adjusting_method string,
	description string, name string, employee_name string, array_day_start_end []day_start_end) []journal_tag {
	array_day_start_end = check_the_params(entry_expair, adjusting_method, date, array_of_entry, array_day_start_end)
	array_of_entry = group_by_account_and_barcode(array_of_entry)
	array_of_entry = remove_zero_values(array_of_entry)
	find_barcode(array_of_entry)
	array_of_entry = s.auto_completion_the_entry(array_of_entry, auto_completion)
	array_of_entry = s.auto_completion_the_invoice_discount(auto_completion, array_of_entry)
	array_of_entry = group_by_account_and_barcode(array_of_entry)
	array_of_entry = remove_zero_values(array_of_entry)
	s.can_the_account_be_negative(array_of_entry)
	debit_entries, credit_entries := s.check_debit_equal_credit(array_of_entry, false)
	simple_entries := s.convert_to_simple_entry(debit_entries, credit_entries)
	var all_array_to_insert []journal_tag
	for _, simple_entry := range simple_entries {
		array_to_insert := insert_to_journal_tag(simple_entry, date, entry_expair, description, name, employee_name)
		if is_in(adjusting_method, depreciation_methods[:]) {
			adjusted_array_to_insert := adjuste_the_array(entry_expair, date, array_day_start_end, array_to_insert, adjusting_method, description, name, employee_name)
			adjusted_array_to_insert = transpose(adjusted_array_to_insert)
			array_to_insert = unpack_the_array(array_to_insert, adjusted_array_to_insert)
		}
		all_array_to_insert = append(all_array_to_insert, array_to_insert...)
	}
	s.insert_to_database(all_array_to_insert, insert, insert, insert)
	return all_array_to_insert
}

func (s Financial_accounting) financial_statements(start_date, end_date time.Time, periods int, names []string, in_names bool) ([]map[string]map[string]map[string]map[string]map[string]float64, []financial_analysis_statement, []journal_tag) {
	check_dates(start_date, end_date)
	days := int(end_date.Sub(start_date).Hours() / 24)
	var journal []journal_tag
	rows, _ := db.Query("select * from journal order by date,entry_number")
	for rows.Next() {
		var entry journal_tag
		rows.Scan(&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.price, &entry.quantity, &entry.barcode, &entry.entry_expair, &entry.description, &entry.name, &entry.employee_name, &entry.entry_date, &entry.reverse)
		journal = append(journal, entry)
	}
	statements := []map[string]map[string]map[string]map[string]map[string]float64{}
	for a := 0; a < periods; a++ {
		flow_statement, nan_flow_statement := s.statement(journal, start_date.AddDate(0, 0, -days*a), end_date.AddDate(0, 0, -days*a))
		statement := combine_statements(flow_statement, nan_flow_statement)
		statement = s.sum_1st_column(statement)
		statement = s.sum_2nd_column(statement)
		sum_3rd_column(statement, []string{}, []string{}, "all", false)
		sum_3rd_column(statement, names, []string{"all"}, "names", in_names)
		vertical_analysis(statement, float64(days))
		statements = append(statements, statement)
	}
	var all_analysis []financial_analysis_statement
	for _, statement_current := range statements {
		horizontal_analysis(statement_current, statements[periods-1])
		s.prepare_statement(statement_current)
		calculate_price(statement_current)
		analysis := s.analysis(statement_current)
		all_analysis = append(all_analysis, analysis)
	}
	return statements, all_analysis, journal
}

func (s Financial_accounting) statement_filter(all_financial_statements []map[string]map[string]map[string]map[string]map[string]float64, account_flow_slice, account_slice, name_slice, vpq_slice, number_slice []string,
	in_account_flow_slice, in_account_slice, in_name_slice, in_vpq_slice, in_number_slice bool) [][]filtered_statement {
	var all_statements_struct [][]filtered_statement
	for _, statement := range all_financial_statements {
		var statement_struct []filtered_statement
		for key_account_flow, map_account_flow := range statement {
			if is_in(key_account_flow, account_flow_slice) == in_account_flow_slice {
				for key_account, map_account := range map_account_flow {
					if is_in(key_account, account_slice) == in_account_slice {
						for key_name, map_name := range map_account {
							if is_in(key_name, name_slice) == in_name_slice {
								for key_vpq, map_vpq := range map_name {
									if is_in(key_vpq, vpq_slice) == in_vpq_slice {
										for key_number, number := range map_vpq {
											if is_in(key_number, number_slice) == in_number_slice {
												statement_struct = append(statement_struct, filtered_statement{key_account_flow, key_account, key_name, key_vpq, key_number, number})
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		var indexa int
		for _, a := range s.accounts {
			for indexb, b := range statement_struct {
				if a.name == b.key_account {
					statement_struct[indexa], statement_struct[indexb] = statement_struct[indexb], statement_struct[indexa]
					indexa++
					break
				}
			}
		}
		all_statements_struct = append(all_statements_struct, statement_struct)
	}
	return all_statements_struct
}

func (s Financial_accounting) can_the_account_be_negative(array_of_entry []Account_value_quantity_barcode) {
	for _, entry := range array_of_entry {
		if !(s.is_father(s.equity, entry.Account) && s.is_credit(entry.Account)) {
			var account_balance float64
			db.QueryRow("select sum(value) from journal where account=? and date<?", entry.Account, Now.String()).Scan(&account_balance)
			if account_balance+entry.value < 0 {
				log.Panic("you cant enter ", entry, " because you have ", account_balance, " and that will make the balance of ", entry.Account, " negative ", account_balance+entry.value, " and that you just can do it in equity_normal accounts not other accounts")
			}
		}
	}
}

func (s Financial_accounting) auto_completion_the_invoice_discount(auto_completion bool, array_of_entry []Account_value_quantity_barcode) []Account_value_quantity_barcode {
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
	return array_of_entry
}

func (s Financial_accounting) auto_completion_the_entry(array_of_entry []Account_value_quantity_barcode, auto_completion bool) []Account_value_quantity_barcode {
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
	return array_of_entry
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

func (s Financial_accounting) reverse_entry(entry_number uint, employee_name string) {
	var array_of_entry_to_reverse, array_of_journal_tag []journal_tag
	rows, _ := db.Query("select * from journal where entry_number=? order by date", entry_number)
	for rows.Next() {
		var tag journal_tag
		rows.Scan(&tag.date, &tag.entry_number, &tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode, &tag.entry_expair, &tag.description, &tag.name, &tag.employee_name, &tag.entry_date, &tag.reverse)
		array_of_journal_tag = append(array_of_journal_tag, tag)
	}
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
				entry.value *= -1
				entry.quantity *= -1
				entry.entry_expair = time.Time{}.String()
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
	entry_number := float64(entry_number())
	for indexa, entry := range array_of_journal_tag {
		entry.entry_number = int(entry_number)
		array_of_journal_tag[indexa].entry_number = int(entry_number)
		entry_number += 0.5
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

func (s Financial_accounting) statement(journal []journal_tag, start_date, end_date time.Time) (map[string]map[string]map[string]map[string]map[string]float64, map[string]map[string]map[string]map[string]float64) {
	var one_simple_entry []journal_tag
	var previous_entry_number int
	var date time.Time
	flow_statement := map[string]map[string]map[string]map[string]map[string]float64{}
	nan_flow_statement := map[string]map[string]map[string]map[string]float64{}
	for _, entry := range journal {
		date = s.parse_date(entry.date)
		if previous_entry_number != entry.entry_number {
			s.sum_flow(date, start_date, one_simple_entry, flow_statement)
			s.sum_values(date, start_date, one_simple_entry, nan_flow_statement)
			one_simple_entry = []journal_tag{}
		}
		if date.Before(end_date) {
			one_simple_entry = append(one_simple_entry, entry)
		}
		previous_entry_number = entry.entry_number
	}
	s.sum_flow(date, start_date, one_simple_entry, flow_statement)
	s.sum_values(date, start_date, one_simple_entry, nan_flow_statement)
	return flow_statement, nan_flow_statement
}

func (s Financial_accounting) sum_values(date, start_date time.Time, one_simple_entry []journal_tag, nan_flow_statement map[string]map[string]map[string]map[string]float64) {
	for _, b := range one_simple_entry {
		map_v1 := initialize_map_3(nan_flow_statement, b.account, b.name, "value")
		map_q1 := initialize_map_3(nan_flow_statement, b.account, b.name, "quantity")
		map_v2 := initialize_map_3(nan_flow_statement, s.retained_earnings, b.name, "value")
		map_q2 := initialize_map_3(nan_flow_statement, s.retained_earnings, b.name, "quantity")
		if date.Before(start_date) {
			switch {
			case s.is_father(s.retained_earnings, b.account) && s.is_credit(b.account):
				map_v2["beginning_balance"] += b.value
				map_q2["beginning_balance"] += b.quantity
			case s.is_father(s.retained_earnings, b.account) && !s.is_credit(b.account):
				map_v2["beginning_balance"] -= b.value
				map_q2["beginning_balance"] -= b.quantity
			default:
				map_v1["beginning_balance"] += b.value
				map_q1["beginning_balance"] += b.quantity
			}
		}
		if date.After(start_date) {
			if b.value >= 0 {
				map_v1["increase"] += math.Abs(b.value)
				map_q1["increase"] += math.Abs(b.quantity)
			} else {
				map_v1["decrease"] += math.Abs(b.value)
				map_q1["decrease"] += math.Abs(b.quantity)
			}
		}
	}
}

func (s Financial_accounting) sum_flow(date, start_date time.Time, one_simple_entry []journal_tag, flow_statement map[string]map[string]map[string]map[string]map[string]float64) {
	for _, a := range one_simple_entry {
		for _, b := range one_simple_entry {
			map_v := initialize_map_4(flow_statement, a.account, b.account, b.name, "value")
			map_q := initialize_map_4(flow_statement, a.account, b.account, b.name, "quantity")
			if date.After(start_date) {
				if b.account == a.account || s.is_credit(b.account) != s.is_credit(a.account) {
					sum_flows(b, 1, map_v, map_q)
				} else {
					sum_flows(b, -1, map_v, map_q)
				}
			}
		}
	}
}

func (s Financial_accounting) prepare_statement(statement map[string]map[string]map[string]map[string]map[string]float64) {
	for key_account_flow, map_account_flow := range statement {
		if key_account_flow == s.cash_and_cash_equivalents {
			for key_account, map_account := range map_account_flow {
				for key_name, map_name := range map_account {
					for key_vpq, map_vpq := range map_name {
						map_vpq1 := initialize_map_4(statement, "financial_statement", key_account, key_name, key_vpq)
						for key_number, number := range map_vpq {
							map_vpq1[key_number] = number
							if !s.is_father(s.income_statement, key_account) {
								map_vpq1["percent"] = statement[s.income_statement][key_account][key_name][key_vpq]["percent"]
							} else {
								map_vpq1["percent"] = statement[s.assets][key_account][key_name][key_vpq]["percent"]
							}
							switch {
							case s.is_father(s.inventory, key_account):
								map_vpq1["turnover"] = statement[s.cost_of_goods_sold][key_account][key_name][key_vpq]["turnover"]
								map_vpq1["turnover_days"] = statement[s.cost_of_goods_sold][key_account][key_name][key_vpq]["turnover_days"]
							case s.is_father(s.assets, key_account):
								map_vpq1["turnover"] = statement[s.sales][key_account][key_name][key_vpq]["turnover"]
								map_vpq1["turnover_days"] = statement[s.sales][key_account][key_name][key_vpq]["turnover_days"]
							}
						}
					}
				}
			}
		}
	}
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

func (s Financial_accounting) check_debit_equal_credit(array_of_entry []Account_value_quantity_barcode, check_one_debit_and_one_credit bool) ([]Account_value_quantity_barcode, []Account_value_quantity_barcode) {
	var debit_entries, credit_entries []Account_value_quantity_barcode
	var zero float64
	for _, entry := range array_of_entry {
		switch s.is_credit(entry.Account) {
		case false:
			zero += entry.value
			if entry.value >= 0 {
				debit_entries = append(debit_entries, entry)
			} else {
				credit_entries = append(credit_entries, entry)
			}
		case true:
			zero -= entry.value
			if entry.value <= 0 {
				debit_entries = append(debit_entries, entry)
			} else {
				credit_entries = append(credit_entries, entry)
			}
		}
	}
	len_debit_entries := len(debit_entries)
	len_credit_entries := len(credit_entries)
	if (len_debit_entries != 1) && (len_credit_entries != 1) {
		log.Panic("should be one credit or one debit in the entry ", array_of_entry)
	}
	if !((len_debit_entries == 1) && (len_credit_entries == 1)) && check_one_debit_and_one_credit {
		log.Panic("should be one credit and one debit in the entry ", array_of_entry)
	}
	if zero != 0 {
		log.Panic(zero, " not equal 0 if the number>0 it means debit overstated else credit overstated debit-credit should equal zero ", array_of_entry)
	}
	return debit_entries, credit_entries
}

func (s Financial_accounting) convert_to_simple_entry(debit_entries, credit_entries []Account_value_quantity_barcode) [][]Account_value_quantity_barcode {
	simple_entries := [][]Account_value_quantity_barcode{}
	for _, debit_entry := range debit_entries {
		for _, credit_entry := range credit_entries {
			simple_entries = append(simple_entries, []Account_value_quantity_barcode{debit_entry, credit_entry})
		}
	}
	for _, a := range simple_entries {
		switch math.Abs(a[0].value) >= math.Abs(a[1].value) {
		case true:
			sign := a[0].value / a[1].value
			price := a[0].value / a[0].quantity
			a[0].value = a[1].value * sign / math.Abs(sign)
			a[0].quantity = a[0].value / price
		case false:
			sign := a[0].value / a[1].value
			price := a[1].value / a[1].quantity
			a[1].value = a[0].value * sign / math.Abs(sign)
			a[1].quantity = a[1].value / price
		}
	}
	return simple_entries
}

func (s Financial_accounting) analysis(statement map[string]map[string]map[string]map[string]map[string]float64) financial_analysis_statement {
	return financial_analysis{
		current_assets:                      statement[s.cash_and_cash_equivalents][s.current_assets]["names"]["value"]["ending_balance"],
		current_liabilities:                 statement[s.cash_and_cash_equivalents][s.current_liabilities]["names"]["value"]["ending_balance"],
		cash:                                statement[s.cash_and_cash_equivalents][s.cash_and_cash_equivalents]["names"]["value"]["ending_balance"],
		short_term_investments:              statement[s.cash_and_cash_equivalents][s.short_term_investments]["names"]["value"]["ending_balance"],
		net_receivables:                     statement[s.cash_and_cash_equivalents][s.receivables]["names"]["value"]["ending_balance"],
		net_credit_sales:                    statement[s.sales][s.receivables]["names"]["value"]["flow"],
		average_net_receivables:             statement[s.cash_and_cash_equivalents][s.receivables]["names"]["value"]["average"],
		cost_of_goods_sold:                  statement[s.cash_and_cash_equivalents][s.cost_of_goods_sold]["names"]["value"]["ending_balance"],
		average_inventory:                   statement[s.cash_and_cash_equivalents][s.inventory]["names"]["value"]["average"],
		net_income:                          statement[s.cash_and_cash_equivalents][s.income_statement]["names"]["value"]["ending_balance"],
		net_sales:                           statement[s.cash_and_cash_equivalents][s.sales]["names"]["value"]["ending_balance"],
		average_assets:                      statement[s.cash_and_cash_equivalents][s.assets]["names"]["value"]["average"],
		average_equity:                      statement[s.cash_and_cash_equivalents][s.equity]["names"]["value"]["average"],
		preferred_dividends:                 0,
		average_common_stockholders_equity:  0,
		market_price_per_shares_outstanding: 0,
		cash_dividends:                      statement[s.cash_and_cash_equivalents][s.dividends]["names"]["value"]["flow"],
		total_debt:                          statement[s.cash_and_cash_equivalents][s.liabilities]["names"]["value"]["ending_balance"],
		total_assets:                        statement[s.cash_and_cash_equivalents][s.assets]["names"]["value"]["ending_balance"],
		ebitda:                              statement[s.cash_and_cash_equivalents][s.ebitda]["names"]["value"]["ending_balance"],
		interest_expense:                    statement[s.cash_and_cash_equivalents][s.interest_expense]["names"]["value"]["ending_balance"],
		weighted_average_common_shares_outstanding: 0,
	}.financial_analysis_statement()
}

func (s Financial_accounting) sum_1st_column(statement map[string]map[string]map[string]map[string]map[string]float64) map[string]map[string]map[string]map[string]map[string]float64 {
	new_statement := map[string]map[string]map[string]map[string]map[string]float64{}
	var flow_accounts []string
	for _, a := range s.accounts {
		for _, b := range s.accounts {
			if s.is_father(a.name, b.name) {
				flow_accounts = append(flow_accounts, b.name)
			}
		}
		for key_account_flow, map_account_flow := range statement {
			if is_in(key_account_flow, flow_accounts) {
				for key_account, map_account := range map_account_flow {
					for key_name, map_name := range map_account {
						for key_vpq, map_vpq := range map_name {
							map_vpq1 := initialize_map_4(new_statement, a.name, key_account, key_name, key_vpq)
							for key_number, number := range map_vpq {
								switch {
								case is_in(key_number, []string{"inflow", "outflow"}):
									if s.is_credit(a.name) == s.is_credit(key_account_flow) {
										map_vpq1[key_number] += number
									} else {
										map_vpq1[key_number] -= number
									}
								default:
									map_vpq1[key_number] = number
								}
							}
						}
					}
				}
			}
		}
		flow_accounts = []string{}
	}
	return new_statement
}

func (s Financial_accounting) sum_2nd_column(statement map[string]map[string]map[string]map[string]map[string]float64) map[string]map[string]map[string]map[string]map[string]float64 {
	new_statement := map[string]map[string]map[string]map[string]map[string]float64{}
	for key_account_flow, map_account_flow := range statement {
		for key_account, map_account := range map_account_flow {
			var last_name string
			key1 := key_account
			for {
				for _, ss := range s.accounts {
					if ss.name == key_account {
						key_account = ss.father
						for key_name, map_name := range map_account {
							for key_vpq, map_vpq := range map_name {
								map_vpq1 := initialize_map_4(new_statement, key_account_flow, ss.name, key_name, key_vpq)
								for key_number, number := range map_vpq {
									switch {
									case !is_in(key_number, []string{"inflow", "outflow"}):
										if s.is_credit(key1) == s.is_credit(ss.name) {
											map_vpq1[key_number] += number
										} else {
											map_vpq1[key_number] -= number
										}
									case key_account_flow != key1:
										map_vpq1[key_number] += number
									case key_account_flow == ss.name:
										new_statement[key_account_flow][key1][key_name][key_vpq][key_number] += number
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
	return new_statement
}

func sum_3rd_column(statement map[string]map[string]map[string]map[string]map[string]float64, names, exempt_names []string, name string, in_names bool) {
	for _, map_account_flow := range statement {
		for _, map_account := range map_account_flow {
			if map_account[name] == nil {
				map_account[name] = map[string]map[string]float64{}
			}
			for key_name, map_name := range map_account {
				var ok bool
				if !is_in(key_name, append(exempt_names, name)) {
					if is_in(key_name, names) == in_names {
						ok = true
					}
					if ok {
						for key_vpq, map_vpq := range map_name {
							if map_account[name][key_vpq] == nil {
								map_account[name][key_vpq] = map[string]float64{}
							}
							for key_number, number := range map_vpq {
								map_account[name][key_vpq][key_number] += number
							}
						}
					}
				}
			}
		}
	}
}

func combine_statements(flow_statement map[string]map[string]map[string]map[string]map[string]float64, nan_flow_statement map[string]map[string]map[string]map[string]float64) map[string]map[string]map[string]map[string]map[string]float64 {
	for key_account_flow, _ := range nan_flow_statement {
		for key_account, map_account := range nan_flow_statement {
			for key_name, map_name := range map_account {
				for key_vpq, map_vpq := range map_name {
					map_vpq1 := initialize_map_4(flow_statement, key_account_flow, key_account, key_name, key_vpq)
					for key_number, _ := range map_vpq {
						map_vpq1[key_number] = map_vpq[key_number]
					}
				}
			}
		}
	}
	return flow_statement
}

func sum_flows(b journal_tag, x float64, map_v, map_q map[string]float64) {
	if b.value*x < 0 {
		map_v["outflow"] += math.Abs(b.value)
		map_q["outflow"] += math.Abs(b.quantity)
	} else {
		map_v["inflow"] += math.Abs(b.value)
		map_q["inflow"] += math.Abs(b.quantity)
	}
}

func ending_balance(statement map[string]map[string]map[string]map[string]map[string]float64, key_account_flow, key_account, key_name, key_vpq string) float64 {
	return statement[key_account_flow][key_account][key_name][key_vpq]["beginning_balance"] + statement[key_account][key_account][key_name][key_vpq]["increase"] - statement[key_account][key_account][key_name][key_vpq]["decrease"]
}

func vertical_analysis(statement map[string]map[string]map[string]map[string]map[string]float64, days float64) {
	for key_account_flow, map_account_flow := range statement {
		for key_account, map_account := range map_account_flow {
			for key_name, map_name := range map_account {
				for key_vpq, map_vpq := range map_name {
					map_vpq["increase_or_decrease"] = map_vpq["increase"] - map_vpq["decrease"]
					map_vpq["ending_balance"] = map_vpq["beginning_balance"] + map_vpq["increase_or_decrease"]
					map_vpq["flow"] = map_vpq["inflow"] - map_vpq["outflow"]
					map_vpq["average"] = (map_vpq["ending_balance"] + map_vpq["beginning_balance"]) / 2
					map_vpq["turnover"] = map_vpq["inflow"] / map_vpq["average"]
					map_vpq["turnover_days"] = days / map_vpq["turnover"]
					map_vpq["growth_ratio"] = map_vpq["ending_balance"] / map_vpq["beginning_balance"]
					map_vpq["percent"] = map_vpq["ending_balance"] / ending_balance(statement, key_account_flow, key_account_flow, key_name, key_vpq)
					map_vpq["name_percent"] = map_vpq["ending_balance"] / ending_balance(statement, key_account_flow, key_account, "all", key_vpq)
				}
			}
		}
	}
}

func horizontal_analysis(statement_current, statement_base map[string]map[string]map[string]map[string]map[string]float64) {
	for key_account_flow, map_account_flow := range statement_current {
		for key_account, map_account := range map_account_flow {
			for key_name, map_name := range map_account {
				for key_vpq, map_vpq := range map_name {
					map_vpq["change_since_base_period"] = map_vpq["ending_balance"] - statement_base[key_account_flow][key_account][key_name][key_vpq]["ending_balance"]
					map_vpq["growth_ratio_to_base_period"] = map_vpq["ending_balance"] / statement_base[key_account_flow][key_account][key_name][key_vpq]["ending_balance"]
				}
			}
		}
	}
}

func calculate_price(statement map[string]map[string]map[string]map[string]map[string]float64) {
	for _, map_account_flow := range statement {
		for _, map_account := range map_account_flow {
			for _, map_name := range map_account {
				if map_name["price"] == nil {
					map_name["price"] = map[string]float64{}
				}
				for _, map_vpq := range map_name {
					for key_number, _ := range map_vpq {
						map_name["price"][key_number] = map_name["value"][key_number] / map_name["quantity"][key_number]
					}
				}
			}
		}
	}
}

func initialize_map_4(m map[string]map[string]map[string]map[string]map[string]float64, a, b, c, d string) map[string]float64 {
	if m[a] == nil {
		m[a] = map[string]map[string]map[string]map[string]float64{}
	}
	if m[a][b] == nil {
		m[a][b] = map[string]map[string]map[string]float64{}
	}
	if m[a][b][c] == nil {
		m[a][b][c] = map[string]map[string]float64{}
	}
	if m[a][b][c][d] == nil {
		m[a][b][c][d] = map[string]float64{}
	}
	return m[a][b][c][d]
}

func initialize_map_3(m map[string]map[string]map[string]map[string]float64, a, b, c string) map[string]float64 {
	if m[a] == nil {
		m[a] = map[string]map[string]map[string]float64{}
	}
	if m[a][b] == nil {
		m[a][b] = map[string]map[string]float64{}
	}
	if m[a][b][c] == nil {
		m[a][b][c] = map[string]float64{}
	}
	return m[a][b][c]
}

func adjuste_the_array(entry_expair time.Time, date time.Time, array_day_start_end []day_start_end, array_to_insert []journal_tag, adjusting_method string, description string, name string, employee_name string) [][]journal_tag {
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
				entry_number:  0,
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
	return adjusted_array_to_insert
}

func check_the_params(entry_expair time.Time, adjusting_method string, date time.Time, array_of_entry []Account_value_quantity_barcode, array_day_start_end []day_start_end) []day_start_end {
	if entry_expair.IsZero() == is_in(adjusting_method, adjusting_methods[:]) {
		log.Panic("check entry_expair => ", entry_expair, " and adjusting_method => ", adjusting_method, " should be in ", adjusting_methods)
	}
	if !entry_expair.IsZero() {
		check_dates(date, entry_expair)
	}
	for _, entry := range array_of_entry {
		if is_in(entry.Account, inventory) && !is_in(adjusting_method, []string{"expire", ""}) {
			log.Panic(entry.Account + " is in inventory you just can use expire or make it empty")
		}
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
	}
	return array_day_start_end
}

func find_barcode(array_of_entry []Account_value_quantity_barcode) {
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
	}
}

func unpack_the_array(array_to_insert []journal_tag, adjusted_array_to_insert [][]journal_tag) []journal_tag {
	array_to_insert = []journal_tag{}
	for _, element := range adjusted_array_to_insert {
		array_to_insert = append(array_to_insert, element...)
	}
	return array_to_insert
}

func insert_to_journal_tag(array_of_entry []Account_value_quantity_barcode, date time.Time, entry_expair time.Time, description string, name string, employee_name string) []journal_tag {
	var array_to_insert []journal_tag
	for _, entry := range array_of_entry {
		price := entry.value / entry.quantity
		if price < 0 {
			log.Panic("the ", entry.value, " and ", entry.quantity, " for ", entry, " should be positive both or negative both")
		}
		array_to_insert = append(array_to_insert, journal_tag{
			date:          date.String(),
			entry_number:  0,
			account:       entry.Account,
			value:         entry.value,
			price:         price,
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
	return array_to_insert
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

func (s Managerial_Accounting) cost_volume_profit_slice() {
	s.check_map_keys()
	s.calculate_cvp_map()
	for _, step := range s.distribution_steps {
		var total_mixed_cost, total_portions_to, total_column_to_distribute float64
		for key_portions_from, portions := range step.from {
			if s.cvp[key_portions_from]["units"] < portions {
				log.Panic(portions, " for ", key_portions_from, " in ", step.from, " is more than ", s.cvp[key_portions_from]["units"])
			}
			total_mixed_cost += portions * s.cvp[key_portions_from]["mixed_cost_per_units"]
			s.cvp[key_portions_from]["fixed_cost"] -= (s.cvp[key_portions_from]["fixed_cost"] / s.cvp[key_portions_from]["units"]) * portions
			s.cvp[key_portions_from]["units"] -= portions
			if s.cvp[key_portions_from]["units"] == 0 {
				s.cvp[key_portions_from]["variable_cost_per_units"] = 0
			}
		}
		for key_portions_to, portions_to := range step.to {
			total_portions_to += portions_to
			total_column_to_distribute += s.cvp[key_portions_to][step.distribution_method]
		}
		for key_portions_to, portions_to := range step.to {
			var total_overhead_cost_to_sum float64
			switch step.distribution_method {
			case "units_gap":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["units_gap"] * s.cvp[key_portions_to]["variable_cost_per_units"]
				s.cvp[key_portions_to]["units"] -= s.cvp[key_portions_to]["units_gap"]
				s.cvp[key_portions_to]["units_gap"] = 0
			case "1":
				total_overhead_cost_to_sum = total_mixed_cost
			case "equally":
				total_overhead_cost_to_sum = float64(len(step.to)) * total_mixed_cost
			case "portions":
				total_overhead_cost_to_sum = portions_to / total_portions_to * total_mixed_cost
			case "units":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["units"] / total_column_to_distribute * total_mixed_cost
			case "variable_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["variable_cost"] / total_column_to_distribute * total_mixed_cost
			case "fixed_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["fixed_cost"] / total_column_to_distribute * total_mixed_cost
			case "mixed_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["mixed_cost"] / total_column_to_distribute * total_mixed_cost
			case "sales":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["sales"] / total_column_to_distribute * total_mixed_cost
			case "profit":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["profit"] / total_column_to_distribute * total_mixed_cost
			case "contribution_margin":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["contribution_margin"] / total_column_to_distribute * total_mixed_cost
			case "percent_from_variable_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["variable_cost"] * portions_to
			case "percent_from_fixed_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["fixed_cost"] * portions_to
			case "percent_from_mixed_cost":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["mixed_cost"] * portions_to
			case "percent_from_sales":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["sales"] * portions_to
			case "percent_from_profit":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["profit"] * portions_to
			case "percent_from_contribution_margin":
				total_overhead_cost_to_sum = s.cvp[key_portions_to]["contribution_margin"] * portions_to
			default:
				log.Panic(step.distribution_method, " is not in [units_gap,1,equally,portions,units,variable_cost,fixed_cost,mixed_cost,sales,profit,contribution_margin,percent_from_variable_cost,percent_from_fixed_cost,percent_from_mixed_cost,percent_from_sales,percent_from_profit,percent_from_contribution_margin]")
			}
			switch step.sales_or_variable_or_fixed {
			case "sales":
				s.cvp[key_portions_to]["sales_per_units"] = ((s.cvp[key_portions_to]["sales_per_units"] * s.cvp[key_portions_to]["units"]) - total_overhead_cost_to_sum) / s.cvp[key_portions_to]["units"]
			case "variable_cost":
				s.cvp[key_portions_to]["variable_cost_per_units"] = ((s.cvp[key_portions_to]["variable_cost_per_units"] * s.cvp[key_portions_to]["units"]) + total_overhead_cost_to_sum) / s.cvp[key_portions_to]["units"]
			case "fixed_cost":
				s.cvp[key_portions_to]["fixed_cost"] += total_overhead_cost_to_sum
			default:
				log.Panic(step.sales_or_variable_or_fixed, " is not in [sales,variable_cost,fixed_cost]")
			}
			for key_name, map_cvp := range s.cvp {
				s.cvp[key_name] = map[string]float64{"units_gap": map_cvp["units_gap"], "units": map_cvp["units"],
					"sales_per_units": map_cvp["sales_per_units"], "variable_cost_per_units": map_cvp["variable_cost_per_units"], "fixed_cost": map_cvp["fixed_cost"]}
			}
			s.calculate_cvp_map()
		}
	}
	s.total_cost_volume_profit()
	s.check_map_keys()
}

func (s Managerial_Accounting) check_map_keys() {
	elements := []string{
		"break_even_in_sales", "break_even_in_units", "sales_per_units",
		"fixed_cost", "break_even_in_units", "contribution_margin_per_units",
		"contribution_margin_per_units", "contribution_margin_ratio", "sales_per_units",
		"contribution_margin", "degree_of_operating_leverage", "profit",
		"variable_cost", "variable_cost_per_units", "units",
		"fixed_cost", "fixed_cost_per_units", "units",
		"mixed_cost", "mixed_cost_per_units", "units",
		"sales", "sales_per_units", "units",
		"profit", "profit_per_units", "units",
		"contribution_margin", "contribution_margin_per_units", "units",
		"mixed_cost", "fixed_cost", "variable_cost",
		"sales", "profit", "mixed_cost",
		"sales", "contribution_margin", "variable_cost",
		"units_gap",
	}
	for _, a := range s.cvp {
		for keyb, _ := range a {
			if !is_in(keyb, elements) {
				log.Panic(keyb, " is not in ", elements)
			}
		}
	}
}

func (s Managerial_Accounting) calculate_cvp_map() {
	for _, i := range s.cvp {
		cost_volume_profit(i)
		_, ok_variable_cost_per_units := i["variable_cost_per_units"]
		if !ok_variable_cost_per_units {
			i["variable_cost_per_units"] = 0
			cost_volume_profit(i)
		}
		_, ok_fixed_cost := i["fixed_cost"]
		if !ok_fixed_cost {
			i["fixed_cost"] = 0
			cost_volume_profit(i)
		}
		_, ok_sales_per_units := i["sales_per_units"]
		if !ok_sales_per_units {
			i["sales_per_units"] = 0
			cost_volume_profit(i)
		}
		_, ok_units := i["units"]
		if !ok_units {
			i["units"] = 1
			cost_volume_profit(i)
		}
	}
}

func (s Managerial_Accounting) total_cost_volume_profit() {
	var units, sales, variable_cost, fixed_cost float64
	for key_name, map_name := range s.cvp {
		if key_name != "total" {
			units += map_name["units"]
			sales += map_name["sales"]
			variable_cost += map_name["variable_cost"]
			fixed_cost += map_name["fixed_cost"]
		}
	}
	s.cvp["total"] = map[string]float64{"units": units, "sales": sales, "variable_cost": variable_cost, "fixed_cost": fixed_cost}
	cost_volume_profit(s.cvp["total"])
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

func cost_volume_profit(m map[string]float64) {
	equations_solver(m, [][]string{
		{"variable_cost", "variable_cost_per_units", "*", "units"},
		{"fixed_cost", "fixed_cost_per_units", "*", "units"},
		{"mixed_cost", "mixed_cost_per_units", "*", "units"},
		{"sales", "sales_per_units", "*", "units"},
		{"profit", "profit_per_units", "*", "units"},
		{"contribution_margin", "contribution_margin_per_units", "*", "units"},
		{"mixed_cost", "fixed_cost", "+", "variable_cost"},
		{"sales", "profit", "+", "mixed_cost"},
		{"contribution_margin", "sales", "-", "variable_cost"},
		{"break_even_in_sales", "break_even_in_units", "*", "sales_per_units"},
		{"break_even_in_units", "contribution_margin_per_units", "/", "fixed_cost"},
		{"contribution_margin_per_units", "contribution_margin_ratio", "*", "sales_per_units"},
		{"contribution_margin", "degree_of_operating_leverage", "*", "profit"},
	})
}

func equations_solver(m map[string]float64, equations [][]string) {
	for a := 0; a <= len(equations); a++ {
		for _, equation := range equations {
			equation_solver(m, equation[0], equation[1], equation[2], equation[3])
		}
	}
}

func equation_solver(m map[string]float64, a, b, sign, c string) {
	switch sign {
	case "+":
		equations_generator(m, a, b, sign, c, m[b]+m[c], m[a]-m[c], m[a]-m[b])
	case "-":
		equations_generator(m, a, b, sign, c, m[b]-m[c], m[a]+m[c], m[b]-m[a])
	case "*":
		equations_generator(m, a, b, sign, c, m[b]*m[c], m[a]/m[c], m[a]/m[b])
	case "/":
		equations_generator(m, a, b, sign, c, m[b]/m[c], m[a]*m[c], m[b]/m[a])
	case "**":
		equations_generator(m, a, b, sign, c, math.Pow(m[b], m[c]), math.Pow(m[a], 1/m[c]), math.Log(m[a])/math.Log(m[b]))
	case "root":
		equations_generator(m, a, b, sign, c, math.Pow(m[b], 1/m[c]), math.Pow(m[a], m[c]), math.Log(m[b])/math.Log(m[a]))
	default:
		log.Panic(sign, " is not in [+-*/**root]")
	}
}

func equations_generator(m map[string]float64, a, b, sign, c string, a_value, b_value, c_value float64) {
	la, oka := m[a]
	lb, okb := m[b]
	lc, okc := m[c]
	var inf bool
	for _, a := range []float64{m[a], m[b], m[c]} {
		if math.IsInf(a, 0) {
			inf = true
		}
	}
	if math.IsNaN(la) {
		m[a] = 0
	}
	if math.IsNaN(lb) {
		m[b] = 0
	}
	if math.IsNaN(lc) {
		m[c] = 0
	}
	if math.IsNaN(a_value) {
		a_value = 0
	}
	switch {
	case !oka && okb && okc:
		m[a] = a_value
		print_equation(m, a, b, sign, c)
	case oka && !okb && okc:
		m[b] = b_value
		print_equation(m, a, b, sign, c)
	case oka && okb && !okc:
		m[c] = c_value
		print_equation(m, a, b, sign, c)
	case oka && okb && okc && math.Round(la*1000)/1000 != math.Round(a_value*1000)/1000 && !inf:
		log.Panic(a, " : ", m[a], " != ", b, " : ", m[b], " ", sign, " ", c, " : ", m[c])
	}
}

func print_equation(m map[string]float64, a, b, sign, c string) {
	fmt.Println(a, " : ", m[a], " = ", b, " : ", m[b], " ", sign, " ", c, " : ", m[c])
}

func main() {
	i := Financial_accounting{
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
			{false, "", "current_assets", "cash_and_cash_equivalents"},
			{false, "", "cash_and_cash_equivalents", "cash"},
			{false, "wma", "current_assets", "short_term_investments"},
			{false, "", "current_assets", "receivables"},
			{false, "wma", "current_assets", "inventory"},
			{false, "wma", "inventory", "book"},
			{false, "wma", "inventory", "book1"},
			{false, "fifo", "inventory", "panadol"},
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
	i.initialize()

	p := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	// entry := i.journal_entry([]Account_value_quantity_barcode{{"cash", 600 - 3.552713678800501e-14, 600 - 3.552713678800501e-14, ""}, {"panadol", 600, -33, ""}, {"sales", 537.1428571428571, 537.1428571428571, ""}}, false, false, Now,
	// 	time.Time{}, "", "", "basma", "hashem", []day_start_end{})

	// i.reverse_entry(8, "hashem")

	// all_financial_statements, _, _ := i.financial_statements(
	// 	time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local),
	// 	time.Date(2022, time.December, 25, 0, 0, 0, 0, time.Local),
	// 	1, []string{}, false)

	// filtered_statement := i.statement_filter(all_financial_statements, []string{"cash", "book"}, []string{"cash", "book"}, []string{"names", "all"}, []string{"value"}, []string{"flow"}, true, true, true, true, true)

	// for _, i := range entry {
	// 	fmt.Fprintln(p, "\t", i.date, "\t", i.entry_number, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.barcode, "\t", i.entry_expair, "\t", i.description, "\t", i.name, "\t", i.employee_name, "\t", i.entry_date, "\t", i.reverse)
	// }

	// fmt.Fprintln(p, "date\t", "entry_number\t", "account\t", "value\t", "price\t", "quantity\t", "barcode\t", "entry_expair\t", "description\t", "name\t", "employee_name\t", "entry_date\t", "reverse")
	// for _, a := range journal_tag {
	// 	fmt.Fprintln(p, a.date, "\t", a.entry_number, "\t", a.account, "\t", a.value, "\t", a.price, "\t", a.quantity, "\t", a.barcode, "\t", a.entry_expair, "\t", a.description, "\t", a.name, "\t", a.employee_name, "\t", a.entry_date, "\t", a.reverse, "\t")
	// }

	// fmt.Fprintln(p, "current_ratio\t", "acid_test\t", "receivables_turnover\t", "inventory_turnover\t", "profit_margin\t", "asset_turnover\t", "return_on_assets\t", "return_on_equity\t", "return_on_common_stockholders_equity\t", "earnings_per_share\t", "price_earnings_ratio\t", "payout_ratio\t", "debt_to_total_assets_ratio\t", "times_interest_earned\t")
	// for _, a := range financial_analysis_statement {
	// 	fmt.Fprintln(p, a.current_ratio, "\t", a.acid_test, "\t", a.receivables_turnover, "\t", a.inventory_turnover, "\t", a.profit_margin, "\t", a.asset_turnover, "\t", a.return_on_assets, "\t", a.return_on_equity, "\t", a.return_on_common_stockholders_equity, "\t", a.earnings_per_share, "\t", a.price_earnings_ratio, "\t", a.payout_ratio, "\t", a.debt_to_total_assets_ratio, "\t", a.times_interest_earned, "\t")
	// }

	// for _, a := range filtered_statement {
	// 	for _, b := range a {
	// 		fmt.Fprintln(p, b.key_account_flow, "\t", b.key_account, "\t", b.key_name, "\t", b.key_vpq, "\t", b.key_number, "\t", b.number)
	// 	}
	// }

	// a1, ok1 := all_financial_statements[0]["panadol"]["cash"]["zaid"]["value"]["inflow"]
	// a2, ok2 := all_financial_statements[0]["cash"]["panadol"]["zaid"]["value"]["outflow"]
	// fmt.Println(a1, ok1)
	// fmt.Println(a2, ok2)

	point := Managerial_Accounting{
		cvp: map[string]map[string]float64{
			"yasa_clinic": {"units": 6 * 4 * 4, "sales_per_units": 70000},
			"electric":    {"units": 80, "fixed_cost": 500000},
			"rent":        {"fixed_cost": 1000000},
		},
		distribution_steps: []one_step_distribution{{"fixed_cost", "units", map[string]float64{"electric": 80}, map[string]float64{"yasa_clinic": 10}}},
	}
	point.cost_volume_profit_slice()
	fmt.Fprintln(p, "key_name", "\t", "units", "\t", "sales_per_units", "\t", "variable_cost_per_units", "\t", "variable_cost", "\t", "fixed_cost", "\t", "mixed_cost", "\t", "mixed_cost_per_units", "\t", "sales", "\t", "profit", "\t", "profit_per_units", "\t", "contribution_margin_per_units", "\t", "contribution_margin", "\t", "contribution_margin_ratio", "\t", "break_even_in_unit", "\t", "break_even_in_sales", "\t", "degree_of_operating_leverage", "\t")
	for key_name, i := range point.cvp {
		fmt.Fprintln(p, key_name, "\t", i["units"], "\t", i["sales_per_units"], "\t", i["variable_cost_per_units"], "\t", i["variable_cost"], "\t", i["fixed_cost"], "\t", i["mixed_cost"], "\t", i["mixed_cost_per_units"], "\t", i["sales"], "\t", i["profit"], "\t", i["profit_per_units"], "\t", i["contribution_margin_per_units"], "\t", i["contribution_margin"], "\t", i["contribution_margin_ratio"], "\t", i["break_even_in_unit"], "\t", i["break_even_in_sales"], "\t", i["degree_of_operating_leverage"], "\t")
	}
	p.Flush()
}
