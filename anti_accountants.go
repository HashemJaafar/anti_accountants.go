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

type directory_account struct {
	directory_no []uint
	account_name string
}

type account_value_price_percent struct {
	account               string
	value, price, percent float64
}

type Financial_accounting struct {
	DriverName, DataSourceName, Database_name string
	Invoice_discounts_tax_list                [][3]float64
	retained_earnings                         [1]directory_account
	Assets_normal, Cash_and_cash_equivalent, Fifo, Lifo, Wma, Assets_contra, Liabilities_normal, Liabilities_contra,
	Equity_normal, Equity_contra, Withdrawals, Sales, Revenues, Discounts, Sales_returns_and_allowances, Expenses []directory_account
	auto_complete_entries [][]account_value_price_percent
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
	directory_no []uint
	account      string
	value, price, quantity, percent,
	base_value, base_price, base_quantity, base_percent,
	changes_since_base_period, current_period_in_relation_to_base_period float64
}

var (
	fifo, lifo, wma, service, inventory, assets_normal, cash_and_cash_equivalent, assets_contra, liabilities_normal, liabilities_contra, equity_normal, equity_contra,
	withdrawals, sales, revenues, sales_returns_and_allowances, expenses, discounts, temporary_debit_accounts, temporary_accounts, debit_accounts, credit_accounts []string
	invoice_discounts_tax_list [][3]float64
	db                         *sql.DB
	all_directory_account      []directory_account
	standard_days              = [7]string{"Saturday", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	adjusting_methods          = [4]string{"linear", "exponential", "logarithmic", "expire"}
	depreciation_methods       = [3]string{"linear", "exponential", "logarithmic"}
	Now                        = time.Now()
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

	fifo = accounts_name(s.Fifo)
	lifo = accounts_name(s.Lifo)
	wma = accounts_name(s.Wma)
	inventory = concat(fifo, lifo, wma).([]string)
	cash_and_cash_equivalent = accounts_name(s.Cash_and_cash_equivalent)
	assets_normal = concat(accounts_name(s.Assets_normal), cash_and_cash_equivalent, inventory).([]string)
	assets_contra = accounts_name(s.Assets_contra)
	liabilities_normal = accounts_name(s.Liabilities_normal)
	liabilities_contra = accounts_name(s.Liabilities_contra)
	equity_normal = concat(accounts_name(s.Equity_normal), accounts_name(s.retained_earnings[:])).([]string)
	equity_contra = accounts_name(s.Equity_contra)
	withdrawals = accounts_name(s.Withdrawals)
	sales = accounts_name(s.Sales)
	revenues = concat(accounts_name(s.Revenues), sales, service).([]string)
	discounts = accounts_name(s.Discounts)
	sales_returns_and_allowances = accounts_name(s.Sales_returns_and_allowances)
	expenses = concat(accounts_name(s.Expenses), sales_returns_and_allowances, discounts).([]string)
	temporary_debit_accounts = concat(withdrawals, expenses).([]string)
	temporary_accounts = concat(temporary_debit_accounts, revenues).([]string)
	debit_accounts = concat(assets_normal, liabilities_contra, equity_contra, temporary_debit_accounts).([]string)
	credit_accounts = concat(assets_contra, liabilities_normal, equity_normal, revenues).([]string)
	all_accounts := concat(debit_accounts, credit_accounts).([]string)
	invoice_discounts_tax_list = s.Invoice_discounts_tax_list

	entry_number := entry_number()
	var array_to_insert []journal_tag
	expair_expenses := journal_tag{Now.String(), entry_number, "expair_expenses", 0, 0, 0, "", time.Time{}.String(),
		"to record the expiry of the goods automatically", "", "", Now.String(), false}
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
	insert_to_database(array_to_insert, true, false, false)
	db.Exec("delete from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", Now.String())
	db.Exec("delete from inventory where quantity=0")

	accounts := column_values("account")
	for _, account := range accounts {
		if !is_in(account, all_accounts) {
			log.Panic(account + " is not on parameters accounts lists")
		}
	}

	all_directory_account = concat(s.retained_earnings[:], s.Assets_normal, s.Cash_and_cash_equivalent, s.Fifo, s.Lifo, s.Wma, s.Assets_contra, s.Liabilities_normal, s.Liabilities_contra,
		s.Equity_normal, s.Equity_contra, s.Withdrawals, s.Sales, s.Revenues, s.Discounts, s.Sales_returns_and_allowances, s.Expenses).([]directory_account)
	all_directorys := concat(accounts_directory(all_directory_account)).([][]uint)
	check_if_duplicates_directory(all_directorys)
	check_if_duplicates(all_accounts)

	for _, directory := range all_directorys {
		l := len(directory)
		if l != 1 {
			ok := false
			for _, d := range all_directorys {
				if reflect.DeepEqual(d, directory[:l-1]) {
					ok = true
					break
				}
			}
			if !ok {
				log.Panic("this directory ", directory, " don't have parent directory like this ", directory[:l-1])
			}
		}
	}
}

func (s Financial_accounting) journal_entry(array_of_entry []Account_value_quantity_barcode, auto_completion bool, entry_to_correct uint, date time.Time, entry_expair time.Time, adjusting_method string,
	description string, name string, employee_name string, array_day_start_end []day_start_end) ([]journal_tag, []account_value_price_percent, time.Time, int) {

	if entry_expair.IsZero() == is_in(adjusting_method, adjusting_methods[:]) {
		log.Panic("check entry_expair => ", entry_expair, " and adjusting_method => ", adjusting_method, " should be in ", adjusting_methods)
	}

	if !entry_expair.IsZero() {
		check_dates(date, entry_expair)
	}

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

	type Account_barcode struct {
		Account, barcode string
	}
	g := map[Account_barcode]*Account_value_quantity_barcode{}
	for _, i := range array_of_entry {
		k := Account_barcode{i.Account, i.barcode}
		sums := g[k]
		if sums == nil {
			sums = &Account_value_quantity_barcode{}
			g[k] = sums
		}
		sums.value += i.value
		sums.quantity += i.quantity
	}
	array_of_entry = []Account_value_quantity_barcode{}
	for k, v := range g {
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{k.Account, v.value, v.quantity, k.barcode})
	}

	var array_of_entry_to_reverse []journal_tag
	if entry_to_correct != 0 {
		rows, _ := db.Query("select * from journal where entry_number=? and date<? and reverse=false", entry_to_correct, Now.String())
		for rows.Next() {
			var tag journal_tag
			rows.Scan(&tag.date, &tag.entry_number, &tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode, &tag.entry_expair, &tag.description, &tag.name, &tag.employee_name, &tag.entry_date, &tag.reverse)
			array_of_entry_to_reverse = append(array_of_entry_to_reverse, tag)
		}
		if len(array_of_entry_to_reverse) == 0 {
			log.Panic("you can't reverse entry number ", entry_to_correct)
		}
		reverse_entry_number := entry_number()
		for index, entry := range array_of_entry_to_reverse {
			array_of_entry_to_reverse[index].date = Now.String()
			array_of_entry_to_reverse[index].entry_number = reverse_entry_number
			array_of_entry_to_reverse[index].value *= -1
			array_of_entry_to_reverse[index].quantity *= -1
			array_of_entry_to_reverse[index].entry_expair = entry_expair.String()
			array_of_entry_to_reverse[index].description = "(reverse entry for entry number " + strconv.Itoa(entry.entry_number) + " entered by " + entry.employee_name + " and revised by " + employee_name + ")"
			array_of_entry_to_reverse[index].employee_name = employee_name
			array_of_entry_to_reverse[index].entry_date = Now.String()
			if is_in(entry.account, inventory) {
				weighted_average([]string{entry.account})
			}
		}
		db.Exec("update journal set reverse=True where entry_number=?", entry_to_correct)
		db.Exec("delete from journal where reverse=True and date>?", Now.String())
	}

	var costs float64
	for index, entry := range array_of_entry {
		is_inventory := is_in(entry.Account, inventory) && entry.quantity < 0
		if is_inventory {
			if is_in(entry.Account, fifo) || is_in(entry.Account, wma) {
				costs = cost_flow(entry.Account, entry.quantity, entry.barcode, "asc", false)
			} else if is_in(entry.Account, lifo) {
				costs = cost_flow(entry.Account, entry.quantity, entry.barcode, "desc", false)
			}
			array_of_entry[index] = Account_value_quantity_barcode{entry.Account, -costs, entry.quantity, entry.barcode}
		}
		if auto_completion {
			for _, complement := range s.auto_complete_entries {
				if complement[0].account == entry.Account {
					if !is_inventory {
						array_of_entry[index] = Account_value_quantity_barcode{complement[0].account, complement[0].price * entry.quantity, entry.quantity, ""}
					}
					for _, i := range complement[1:] {
						if i.value == 0 {
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, i.price * entry.quantity * i.percent, entry.quantity * i.percent, ""})
						} else {
							array_of_entry = append(array_of_entry, Account_value_quantity_barcode{i.account, i.value, i.value / i.price, ""})
						}
					}
					break
				}
			}
		}
	}
	if auto_completion {
		var total_invoice_before_invoice_discount float64
		for _, entry := range array_of_entry {
			if is_in(entry.Account, revenues) {
				total_invoice_before_invoice_discount += entry.value
			} else if is_in(entry.Account, discounts) {
				total_invoice_before_invoice_discount -= entry.value
			}
		}
		var discount float64
		var tax float64
		for _, i := range invoice_discounts_tax_list {
			if total_invoice_before_invoice_discount >= i[0] {
				discount = i[1]
				tax = i[2]
			}
		}
		invoice_discount := discount_tax_calculator(total_invoice_before_invoice_discount, discount)
		invoice_tax := discount_tax_calculator(total_invoice_before_invoice_discount, tax)
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{"invoice discount", invoice_discount, 1, ""})
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{"invoice tax", invoice_tax, 1, ""})
		array_of_entry = append(array_of_entry, Account_value_quantity_barcode{"tax", invoice_tax, 1, ""})
	}

	var index int
	for index < len(array_of_entry) {
		if array_of_entry[index].value == 0 || array_of_entry[index].quantity == 0 {
			// fmt.Println(array_of_entry[index], " is removed because one of the values is 0")
			array_of_entry = append(array_of_entry[:index], array_of_entry[index+1:]...)
		} else {
			index++
		}
	}
	var zero float64
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
		if is_in(entry.Account, debit_accounts) {
			zero += entry.value
		} else if is_in(entry.Account, credit_accounts) {
			zero -= entry.value
		} else {
			log.Panic(entry.Account, " is not on parameters accounts lists")
		}
	}
	if zero != 0 {
		log.Panic(zero, " not equal 0 if the number>0 it means debit overstated else credit overstated debit-credit should equal zero ", array_of_entry)
	}

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

	var invoice []account_value_price_percent
	for _, entry := range array_to_insert {
		switch {
		case is_in(entry.account, assets_normal) && !is_in(entry.account, inventory) && entry.value > 0:
			invoice = append(invoice, account_value_price_percent{"total", entry.value, entry.price, entry.quantity})
		case is_in(entry.account, discounts):
			invoice = append(invoice, account_value_price_percent{"total discounts", entry.value, entry.price, entry.quantity})
		case is_in(entry.account, sales):
			invoice = append(invoice, account_value_price_percent{entry.account, entry.value, entry.price, entry.quantity})
		}
	}
	m := map[string]*account_value_price_percent{}
	for _, i := range invoice {
		k := i.account
		sums := m[k]
		if sums == nil {
			sums = &account_value_price_percent{}
			m[k] = sums
		}
		sums.value += i.value
		sums.percent = i.percent
	}
	invoice = []account_value_price_percent{}
	for k, v := range m {
		invoice = append(invoice, account_value_price_percent{k, v.value, v.value / v.percent, v.percent})
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

	array_to_insert = append(array_of_entry_to_reverse, array_to_insert...)
	insert_to_database(array_to_insert, true, true, true)
	return array_to_insert, invoice, Now, entry_number
}

func (s Financial_accounting) financial_statements(start_base_date, end_base_date, start_date, end_date time.Time, remove_empties bool) ([]statement, []statement, []statement) {
	check_dates(start_date, end_date)
	check_dates(end_base_date, start_date)
	check_dates(start_base_date, end_base_date)
	rows, _ := db.Query("select date,entry_number,account,value,quantity from journal order by date")
	var journal []journal_tag
	for rows.Next() {
		var tag journal_tag
		rows.Scan(&tag.date, &tag.entry_number, &tag.account, &tag.value, &tag.quantity)
		journal = append(journal, tag)
	}

	var cash, cash_base []journal_tag
	var previous_date string
	var ok, ok_base bool
	var assets, assets_base, net_sales, net_sales_base, cash_increase, cash_increase_base float64
	retained_earnings := statement{directory_no: s.retained_earnings[0].directory_no, account: s.retained_earnings[0].account_name}
	journal_map := map[string]*statement{}
	income_map := map[string]*statement{}
	cash_map := map[string]*statement{}
	for _, entry := range journal {
		key_journal := entry.account
		sum_journal := journal_map[key_journal]
		if sum_journal == nil {
			sum_journal = &statement{}
			journal_map[key_journal] = sum_journal
		}
		key_income := entry.account
		sum_income := income_map[key_income]
		if sum_income == nil {
			sum_income = &statement{}
			income_map[key_income] = sum_income
		}

		date, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 +03 m=+0.999999999", entry.date)
		if err != nil {
			date, _ = time.Parse("2006-01-02 15:04:05.999999999 -0700 +03", entry.date)
		}
		if previous_date != entry.date {
			if ok_base {
				for _, entry := range cash_base {
					key_cash := entry.account
					sum_cash := cash_map[key_cash]
					if sum_cash == nil {
						sum_cash = &statement{}
						cash_map[key_cash] = sum_cash
					}
					if is_in(entry.account, credit_accounts) {
						sum_cash.base_value += entry.value
						sum_cash.base_quantity += entry.quantity
						cash_increase_base += entry.value
					} else {
						sum_cash.base_value -= entry.value
						sum_cash.base_quantity -= entry.quantity
						cash_increase_base -= entry.value
					}
				}
			}
			if ok {
				for _, entry := range cash {
					key_cash := entry.account
					sum_cash := cash_map[key_cash]
					if sum_cash == nil {
						sum_cash = &statement{}
						cash_map[key_cash] = sum_cash
					}
					if is_in(entry.account, credit_accounts) {
						sum_cash.value += entry.value
						sum_cash.quantity += entry.quantity
						cash_increase += entry.value
					} else {
						sum_cash.value -= entry.value
						sum_cash.quantity -= entry.quantity
						cash_increase -= entry.value
					}
				}
			}
			cash_base = []journal_tag{}
			cash = []journal_tag{}
			ok_base = false
			ok = false
		}
		previous_date = entry.date

		if date.Before(start_base_date) {
			if is_in(entry.account, revenues) {
				retained_earnings.base_value += entry.value
			} else if is_in(entry.account, temporary_debit_accounts) {
				retained_earnings.base_value -= entry.value
			}
		}
		if date.After(start_base_date) && date.Before(end_base_date) {
			if is_in(entry.account, cash_and_cash_equivalent) {
				ok_base = true
			} else {
				cash_base = append(cash_base, entry)
			}
			if is_in(entry.account, revenues) {
				sum_income.base_value += entry.value
				sum_income.base_quantity += entry.quantity
			} else if is_in(entry.account, expenses) {
				sum_income.base_value += entry.value
				sum_income.base_quantity += entry.quantity
			}
			if is_in(entry.account, sales) {
				net_sales_base += entry.value
			} else if is_in(entry.account, sales_returns_and_allowances) {
				net_sales_base -= entry.value
			}
		}
		if date.Before(end_base_date) {
			sum_journal.base_value += entry.value
			sum_journal.base_quantity += entry.quantity
			if is_in(entry.account, assets_normal) {
				assets += entry.value
			} else if is_in(entry.account, assets_contra) {
				assets -= entry.value
			}
		}
		if date.Before(start_date) {
			if is_in(entry.account, revenues) {
				retained_earnings.value += entry.value
			} else if is_in(entry.account, temporary_debit_accounts) {
				retained_earnings.value -= entry.value
			}
		}
		if date.After(start_date) && date.Before(end_date) {
			if is_in(entry.account, cash_and_cash_equivalent) {
				ok = true
			} else {
				cash = append(cash, entry)
			}
			if is_in(entry.account, revenues) {
				sum_income.value += entry.value
				sum_income.quantity += entry.quantity
			} else if is_in(entry.account, expenses) {
				sum_income.value += entry.value
				sum_income.quantity += entry.quantity
			}
			if is_in(entry.account, sales) {
				net_sales += entry.value
			} else if is_in(entry.account, sales_returns_and_allowances) {
				net_sales -= entry.value
			}
		}
		if date.Before(end_date) {
			sum_journal.value += entry.value
			sum_journal.quantity += entry.quantity
			if is_in(entry.account, assets_normal) {
				assets += entry.value
			} else if is_in(entry.account, assets_contra) {
				assets -= entry.value
			}
		}
	}
	balance_sheet := append(prepare_statement(journal_map, assets, assets_base), retained_earnings)
	income_statements := prepare_statement(income_map, net_sales, net_sales_base)
	cash_flow := prepare_statement(cash_map, cash_increase, cash_increase_base)
	return balance_sheet, income_statements, cash_flow
}

func insert_to_database(array_to_insert []journal_tag, insert_into_journal, insert_into_inventory, inventory_flow bool) {
	for _, entry := range array_to_insert {
		if insert_into_journal {
			db.Exec("insert into journal(date,entry_number,account,value,price,quantity,barcode,entry_expair,description,name,employee_name,entry_date,reverse) values (?,?,?,?,?,?,?,?,?,?,?,?,?)",
				&entry.date, &entry.entry_number, &entry.account, &entry.value, &entry.price, &entry.quantity, &entry.barcode,
				&entry.entry_expair, &entry.description, &entry.name, &entry.employee_name, &entry.entry_date, &entry.reverse)
		}
		if is_in(entry.account, inventory) {
			if entry.quantity > 0 {
				if insert_into_inventory {
					db.Exec("insert into inventory(date,account,price,quantity,barcode,entry_expair,name,employee_name,entry_date)values (?,?,?,?,?,?,?,?,?)",
						&entry.date, &entry.account, &entry.price, &entry.quantity, &entry.barcode, &entry.entry_expair, &entry.name, &entry.employee_name, &entry.entry_date)
				}
			} else {
				if inventory_flow {
					switch {
					case is_in(entry.account, fifo):
						cost_flow(entry.account, entry.quantity, entry.barcode, "asc", true)
					case is_in(entry.account, lifo):
						cost_flow(entry.account, entry.quantity, entry.barcode, "desc", true)
					case is_in(entry.account, wma):
						weighted_average([]string{entry.account})
						cost_flow(entry.account, entry.quantity, entry.barcode, "asc", true)
					}
				}
			}
		}
	}
}

func cost_flow(account string, quantity float64, barcode string, order_by_date_asc_or_desc string, insert bool) float64 {
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

func prepare_statement(journal_map map[string]*statement, amount, amount_base float64) []statement {
	statement_sheet := []statement{}
	for key, v := range journal_map {
		var directory []uint
		for _, i := range all_directory_account {
			if key == i.account_name {
				directory = i.directory_no
				break
			}
		}
		statement_sheet = append(statement_sheet, statement{
			directory_no:              directory,
			account:                   key,
			value:                     v.value,
			price:                     v.value / v.quantity,
			quantity:                  v.quantity,
			percent:                   v.value / amount,
			base_value:                v.base_value,
			base_price:                v.base_value / v.base_quantity,
			base_quantity:             v.base_quantity,
			base_percent:              v.base_value / amount_base,
			changes_since_base_period: v.value - v.base_value,
			current_period_in_relation_to_base_period: v.value / v.base_value,
		})
	}
	return statement_sheet
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

func column_values(column string) []string {
	results, err := db.Query("select " + column + " from journal")
	error_fatal(err)
	column_values := []string{}
	for results.Next() {
		var tag string
		results.Scan(&tag)
		column_values = append(column_values, tag)
	}
	return column_values
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

func check_if_duplicates_directory(slice_of_elements [][]uint) {
	var set_of_elems, duplicated_element [][]uint
	for _, element := range slice_of_elements {
		for _, b := range set_of_elems {
			if reflect.DeepEqual(b, element) {
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

func accounts_name(args []directory_account) []string {
	accounts_slice := []string{}
	for _, i := range args {
		accounts_slice = append(accounts_slice, i.account_name)
	}
	return accounts_slice
}

func accounts_directory(args []directory_account) [][]uint {
	accounts_slice := [][]uint{}
	for _, i := range args {
		accounts_slice = append(accounts_slice, i.directory_no)
	}
	return accounts_slice
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

func main() {
	v := Financial_accounting{
		DriverName:                   "mysql",
		DataSourceName:               "hashem:hashem@tcp(localhost)/",
		Database_name:                "acc",
		Invoice_discounts_tax_list:   [][3]float64{{10, -5, 0.01}, {0, 0, 0}},
		retained_earnings:            [1]directory_account{{[]uint{4, 4}, "retained_earnings"}},
		Assets_normal:                []directory_account{{[]uint{1, 1}, "office equipment"}, {[]uint{1}, "Assets_normal"}, {[]uint{1, 3}, "inventory"}},
		Cash_and_cash_equivalent:     []directory_account{{[]uint{1, 8}, "cash"}},
		Fifo:                         []directory_account{{[]uint{1, 3, 1}, "book1"}},
		Lifo:                         []directory_account{{[]uint{1, 3, 2}, "book2"}},
		Wma:                          []directory_account{{[]uint{1, 3, 3}, "book"}, {[]uint{1, 3, 4}, "book3"}},
		Assets_contra:                []directory_account{},
		Liabilities_normal:           []directory_account{{[]uint{2, 1}, "tax"}, {[]uint{2}, "Liabilities"}},
		Liabilities_contra:           []directory_account{},
		Equity_normal:                []directory_account{},
		Equity_contra:                []directory_account{},
		Withdrawals:                  []directory_account{},
		Sales:                        []directory_account{{[]uint{4, 1}, "revenue of service revenue"}, {[]uint{4, 2}, "revenue of book"}, {[]uint{4, 3, 4}, "service revenue"}},
		Revenues:                     []directory_account{{[]uint{4, 3}, "service"}, {[]uint{4}, "revenue"}},
		Discounts:                    []directory_account{{[]uint{3, 1}, "discount of service revenue"}, {[]uint{3, 6}, "discount of book"}, {[]uint{3, 7}, "invoice discount"}},
		Sales_returns_and_allowances: []directory_account{},
		Expenses:                     []directory_account{{[]uint{3, 4}, "tax of service revenue"}, {[]uint{3, 2}, "expair_expenses"}, {[]uint{3, 3}, "cost of book"}, {[]uint{3, 5}, "tax of book"}, {[]uint{3, 8}, "invoice tax"}, {[]uint{3}, "expenses"}},
		auto_complete_entries: [][]account_value_price_percent{
			{{"service revenue", 0, 10, 1}, {"tax of service revenue", 1, 1, 0}, {"tax", 1, 1, 0}},
			{{"book", 0, 0, 1}, {"revenue of book", 0, 10, -1}, {"cost of book", 0, 0, -1}, {"tax of book", 1, 1, 0}, {"discount of book", 1, 1, 0}},
		},
	}
	v.initialize()
	// entry, invoice, t, entry_number := v.journal_entry([]Account_value_quantity_barcode{{"cash", 100000, 100000, ""}, {"service revenue", 50, 10000, ""}}, true, 0 /*uint(entry_number())-1*/, time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local),
	// 	time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local), "linear", "", "yasa", "hashem", []day_start_end{})

	// fmt.Println(invoice, t, entry_number)
	// r := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	// for _, i := range entry {
	// 	fmt.Fprintln(r, "\t", i.date, "\t", i.entry_number, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.barcode, "\t", i.entry_expair, "\t", i.description, "\t", i.name, "\t", i.employee_name, "\t", i.entry_date, "\t", i.reverse)
	// }
	// r.Flush()

	balance_sheet, income_statements, cash_flow := v.financial_statements(
		time.Date(2019, time.January, 1, 0, 0, 0, 0, time.Local),
		time.Date(2019, time.January, 1, 0, 0, 0, 0, time.Local),
		time.Date(2019, time.January, 1, 0, 0, 0, 0, time.Local),
		time.Date(2023, time.January, 1, 0, 0, 0, 0, time.Local),
		true)

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, "####################################################################### balance_sheet #######################################################################")
	for _, i := range balance_sheet {
		fmt.Fprintln(w, i.directory_no, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.base_value, "\t", i.base_price, "\t", i.base_quantity, "\t", i.base_percent, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	}
	fmt.Fprintln(w, "##################################################################### income_statements #####################################################################")
	for _, i := range income_statements {
		fmt.Fprintln(w, i.directory_no, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.base_value, "\t", i.base_price, "\t", i.base_quantity, "\t", i.base_percent, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	}
	fmt.Fprintln(w, "######################################################################### cash_flow #########################################################################")
	for _, i := range cash_flow {
		fmt.Fprintln(w, i.directory_no, "\t", i.account, "\t", i.value, "\t", i.price, "\t", i.quantity, "\t", i.percent, "\t", i.base_value, "\t", i.base_price, "\t", i.base_quantity, "\t", i.base_percent, "\t", i.changes_since_base_period, "\t", i.current_period_in_relation_to_base_period, "\t")
	}
	w.Flush()
}
