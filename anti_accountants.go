package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tobgu/qframe"
	qsql "github.com/tobgu/qframe/config/sql"
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

type directory_account_price_discount_tax struct {
	directory_no         []uint
	Account              string
	Price, Discount, Tax float64
}

type directory_account struct {
	directory_no []uint
	account_name string
}

type Financial_accounting struct {
	Company_name               string
	Discount                   float64
	Invoice_discounts_tax_list [][3]float64
	Fifo, Lifo, Wma, Service   []directory_account_price_discount_tax
	Assets_normal, Cash_and_cash_equivalent, Assets_contra, Liabilities_normal, Liabilities_contra,
	Equity_normal, Equity_contra, Withdrawals, Revenues, Discounts, Expenses []directory_account
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

type invoice_tag struct {
	account              string
	value_after_discount float64
	value                float64
	price                float64
	total_discount       float64
	discount             float64
	quantity             float64
}

type statement struct {
	directory_no []uint
	Account      string
	value, price, quantity, analysis,
	budget_value, budget_price, budget_quantity, budget_analysis,
	difference float64
}

var (
	fifo, lifo, wma, service, inventory, assets_normal, cash_and_cash_equivalent, assets_contra, liabilities_normal, liabilities_contra, equity_normal, equity_contra,
	withdrawals, revenues, expenses, discounts, temporary_debit_accounts, temporary_accounts, debit_accounts, credit_accounts []string
	invoice_discounts_tax_list [][3]float64
	db                         *sql.DB
	err                        error
	price_discount_tax         []directory_account_price_discount_tax
	standard_days              = [7]string{"Saturday", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	adjusting_methods          = [4]string{"linear", "exponential", "logarithmic", "expire"}
	depreciation_methods       = [3]string{"linear", "exponential", "logarithmic"}
	start_date, end_date       time.Time
	now                        = time.Now()
	date                       = now
)

func (s Financial_accounting) initialize() {
	// cfg := mysql.Config{
	// 	User:   "hashem",
	// 	Passwd: "hashem",
	// 	Net:    "tcp",
	// 	Addr:   "localhost",
	// 	DBName: s.Company_name,
	// }
	db, _ = sql.Open("mysql", "hashem:hashem@tcp(localhost)/")
	err = db.Ping()
	error_fatal(err)
	db.Exec("create database if not exists " + s.Company_name)
	_, err = db.Exec("USE " + s.Company_name)
	error_fatal(err)
	db.Exec("create table if not exists journal (date text,entry_number integer,account text,value real,price real,quantity real,barcode text,entry_expair text,description text,name text,employee_name text,entry_date text,reverse bool)")
	db.Exec("create table if not exists inventory (date text,account text,price real,quantity real,barcode text,entry_expair text,name text,employee_name text,entry_date text)")
	// defer db.Close()s
	// defer insert.Close()

	price_discount_tax = concat(s.Fifo, s.Lifo, s.Wma, s.Service).([]directory_account_price_discount_tax)
	for index, i := range price_discount_tax {
		price_discount_tax[index].Discount = discount_tax_calculator(i.Price, i.Discount)
		price_discount_tax[index].Tax = discount_tax_calculator(i.Price, i.Tax)
	}
	if s.Discount != 0 {
		for index, i := range price_discount_tax {
			price_discount_tax[index].Discount = discount_tax_calculator(i.Price, s.Discount)
		}
	}

	fifo = accounts_slice(s.Fifo)
	lifo = accounts_slice(s.Lifo)
	wma = accounts_slice(s.Wma)
	service = accounts_slice(s.Service)
	inventory = concat(fifo, lifo, wma).([]string)
	cash_and_cash_equivalent = accounts_name(s.Cash_and_cash_equivalent)
	assets_normal = concat(accounts_name(s.Assets_normal), cash_and_cash_equivalent, inventory).([]string)
	assets_contra = accounts_name(s.Assets_contra)
	liabilities_normal = accounts_name(s.Liabilities_normal)
	liabilities_contra = accounts_name(s.Liabilities_contra)
	equity_normal = accounts_name(s.Equity_normal)
	equity_contra = accounts_name(s.Equity_contra)
	withdrawals = accounts_name(s.Withdrawals)
	revenues = accounts_name(s.Revenues)
	discounts = accounts_name(s.Discounts)
	expenses = concat(accounts_name(s.Expenses), discounts).([]string)
	temporary_debit_accounts = concat(withdrawals, expenses).([]string)
	temporary_accounts = concat(temporary_debit_accounts, revenues).([]string)
	debit_accounts = concat(assets_normal, liabilities_contra, equity_contra, temporary_debit_accounts).([]string)
	credit_accounts = concat(assets_contra, liabilities_normal, equity_normal, revenues).([]string)
	all_accounts := concat(debit_accounts, credit_accounts).([]string)
	invoice_discounts_tax_list = s.Invoice_discounts_tax_list

	entry_number := entry_number()
	var array_to_insert []journal_tag
	expair_expenses := journal_tag{now.String(), entry_number, "expair_expenses", 0, 0, 0, "", time.Time{}.String(),
		"to record the expiry of the goods automatically", "", "", now.String(), false}
	expair_goods, _ := db.Query("select account,price*quantity*-1,price,quantity*-1,barcode from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", now.String())
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
	db.Exec("delete from inventory where entry_expair<? and entry_expair!='0001-01-01 00:00:00 +0000 UTC'", now.String())
	db.Exec("delete from inventory where quantity=0")

	accounts := column_values("account")
	for _, account := range accounts {
		if !is_in(account, all_accounts) {
			log.Fatal(account + " is not on parameters accounts lists")
		}
	}

	all_directorys := concat(
		accounts_directory(s.Fifo),
		accounts_directory(s.Lifo),
		accounts_directory(s.Wma),
		accounts_directory(s.Service),
		accounts_directory1(s.Assets_normal),
		accounts_directory1(s.Cash_and_cash_equivalent),
		accounts_directory1(s.Assets_contra),
		accounts_directory1(s.Liabilities_normal),
		accounts_directory1(s.Liabilities_contra),
		accounts_directory1(s.Equity_normal),
		accounts_directory1(s.Equity_contra),
		accounts_directory1(s.Withdrawals),
		accounts_directory1(s.Revenues),
		accounts_directory1(s.Discounts),
		accounts_directory1(s.Expenses)).([][]uint)

	check_if_duplicates_directory(all_directorys)
	check_if_duplicates(all_accounts)
}

func journal_entry(array_of_entry []Account_value_quantity_barcode, auto_completion bool, entry_to_correct uint, date time.Time, entry_expair time.Time, adjusting_method string,
	description string, name string, employee_name string, array_day_start_end []day_start_end) ([]journal_tag, []invoice_tag, time.Time, int) {

	if entry_expair.IsZero() == is_in(adjusting_method, adjusting_methods[:]) {
		log.Fatal("check entry_expair => ", entry_expair, " and adjusting_method => ", adjusting_method, " should be in ", adjusting_methods)
	}

	if !entry_expair.IsZero() {
		date, entry_expair = check_dates(dates(date), dates(entry_expair))
	} else {
		date = dates(date)
	}

	for index, entry := range array_of_entry {
		if entry.Account == "" && entry.barcode == "" {
			log.Fatal("can't find the account name if the barcode is empty in ", entry)
		}
		var tag string
		if entry.Account == "" {
			err = db.QueryRow("select account from journal where barcode=? limit 1", entry.barcode).Scan(&tag)
			if err != nil {
				log.Fatal("the barcode is wrong for ", entry)
			}
			array_of_entry[index].Account = tag
		}
		if is_in(entry.Account, inventory) && !is_in(adjusting_method, []string{"expire", ""}) {
			log.Fatal(entry.Account + " is in inventory you just can use expire or make it empty")
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
		rows, _ := db.Query("select * from journal where entry_number=? and date<? and reverse=false", entry_to_correct, now.String())
		for rows.Next() {
			var tag journal_tag
			rows.Scan(&tag.date, &tag.entry_number, &tag.account, &tag.value, &tag.price, &tag.quantity, &tag.barcode, &tag.entry_expair, &tag.description, &tag.name, &tag.employee_name, &tag.entry_date, &tag.reverse)
			array_of_entry_to_reverse = append(array_of_entry_to_reverse, tag)
		}
		if len(array_of_entry_to_reverse) == 0 {
			log.Fatal("you can't reverse entry number ", entry_to_correct)
		}
		reverse_entry_number := entry_number()
		for index, entry := range array_of_entry_to_reverse {
			array_of_entry_to_reverse[index].date = now.String()
			array_of_entry_to_reverse[index].entry_number = reverse_entry_number
			array_of_entry_to_reverse[index].value *= -1
			array_of_entry_to_reverse[index].quantity *= -1
			array_of_entry_to_reverse[index].entry_expair = entry_expair.String()
			array_of_entry_to_reverse[index].description = "(reverse entry for entry number " + strconv.Itoa(entry.entry_number) + " entered by " + entry.employee_name + " and revised by " + employee_name + ")"
			array_of_entry_to_reverse[index].employee_name = employee_name
			array_of_entry_to_reverse[index].entry_date = now.String()
			if is_in(entry.account, inventory) {
				weighted_average([]string{entry.account})
			}
		}
		db.Exec("update journal set reverse=True where entry_number=?", entry_to_correct)
		db.Exec("delete from journal where reverse=True and date>?", now.String())
	}

	if auto_completion {
		var total_invoice_before_invoice_discount float64
		var costs float64
		for index, entry := range array_of_entry {
			quantity := math.Abs(entry.quantity)
			if is_in(entry.Account, inventory) && entry.quantity < 0 {
				if is_in(entry.Account, fifo) || is_in(entry.Account, wma) {
					costs = cost_flow(entry.Account, entry.quantity, entry.barcode, "asc", false)
				} else if is_in(entry.Account, lifo) {
					costs = cost_flow(entry.Account, entry.quantity, entry.barcode, "desc", false)
				} else {
					continue
				}
				array_of_entry[index] = Account_value_quantity_barcode{entry.Account, -costs, -quantity, entry.barcode}
				array_of_entry = append(array_of_entry, Account_value_quantity_barcode{"cost of " + entry.Account, costs, quantity, entry.barcode})
				array_of_entry = append(array_of_entry, price_discount_tax_list(entry.Account, quantity)...)
			} else if is_in(entry.Account, service) {
				array_of_entry = append(array_of_entry[:index], array_of_entry[index+1:]...)
				array_of_entry = append(array_of_entry, price_discount_tax_list(entry.Account, quantity)...)
			}
			if is_in(entry.Account, revenues) {
				total_invoice_before_invoice_discount += entry.value
			}
			if is_in(entry.Account, discounts) {
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
			db.QueryRow("select sum(value) from journal where account=? and date<?", entry.Account, now.String()).Scan(&account_balance)
			if account_balance+entry.value < 0 {
				log.Fatal("you cant enter ", entry, " because you have ", account_balance, " and that will make the balance of ", entry.Account, " negative ", account_balance+entry.value, " and that you just can do it in equity_normal accounts not other accounts")
			}
		}
		price_slice = append(price_slice, entry.value/entry.quantity)
		if price_slice[index] < 0 {
			log.Fatal("the ", entry.value, " and ", entry.quantity, " for ", entry, " should be positive both or negative both")
		}
		if is_in(entry.Account, debit_accounts) {
			zero += entry.value
		} else if is_in(entry.Account, credit_accounts) {
			zero -= entry.value
		} else {
			log.Fatal(entry.Account, " is not on parameters accounts lists")
		}
	}
	if zero != 0 {
		log.Fatal(zero, " not equal 0 if the number>0 it means debit overstated else credit overstated debit-credit should equal zero")
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
			entry_date:    now.String(),
			reverse:       false,
		})
	}

	type key struct {
		account  string
		quantity float64
	}
	var invoice []invoice_tag
	for _, entry := range array_to_insert {
		switch {
		case is_in(entry.account, assets_normal) && !is_in(entry.account, inventory) && entry.value > 0:
			invoice = append(invoice, invoice_tag{entry.account, entry.value, entry.value, entry.price, 0, 0, entry.quantity})
		case entry.account == "invoice discount":
			invoice = append(invoice, invoice_tag{entry.account, -entry.value, 0, 0, entry.value, entry.price, entry.quantity})
		case strings.Contains(entry.account, "discount of "):
			invoice = append(invoice, invoice_tag{strings.Replace(entry.account, "discount of ", "", -1), -entry.value, 0, 0, entry.value, entry.price, entry.quantity})
		case strings.Contains(entry.account, "revenue of "):
			invoice = append(invoice, invoice_tag{strings.Replace(entry.account, "revenue of ", "", -1), entry.value, entry.value, entry.price, 0, 0, entry.quantity})
		}
	}
	m := map[key]*invoice_tag{}
	for _, i := range invoice {
		k := key{i.account, i.quantity}
		sums := m[k]
		if sums == nil {
			sums = &invoice_tag{}
			m[k] = sums
		}
		sums.value_after_discount += i.value_after_discount
		sums.value += i.value
		sums.price += i.price
		sums.total_discount += i.total_discount
		sums.discount += i.discount
	}
	invoice = []invoice_tag{}
	for k, v := range m {
		invoice = append(invoice, invoice_tag{k.account, v.value_after_discount, v.value, v.price, v.total_discount, v.discount, k.quantity})
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
				log.Fatal("error ", element.day, " for ", element, " is not in ", standard_days)
			case element.start_hour < 0:
				log.Fatal("error ", element.start_hour, " for ", element, " is < 0")
			case element.start_hour > 23:
				log.Fatal("error ", element.start_hour, " for ", element, " is > 23")
			case element.start_minute < 0:
				log.Fatal("error ", element.start_minute, " for ", element, " is < 0")
			case element.start_minute > 59:
				log.Fatal("error ", element.start_minute, " for ", element, " is > 59")
			case element.end_hour < 0:
				log.Fatal("error ", element.end_hour, " for ", element, " is < 0")
			case element.end_hour > 23:
				log.Fatal("error ", element.end_hour, " for ", element, " is > 23")
			case element.end_minute < 0:
				log.Fatal("error ", element.end_minute, " for ", element, " is < 0")
			case element.end_minute > 59:
				log.Fatal("error ", element.end_minute, " for ", element, " is > 59")
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
						log.Fatal("the start_hour and start_minute should be smaller than end_hour and end_minute for ", element)
					}
					if previous_end_date.After(start) {
						log.Fatal("the end_hour and end_minute for ", element.day, " should be smaller than start_hour and start_minute for the second ", element)
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
					entry_date:    now.String(),
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
	return array_to_insert, invoice, now, entry_number
}

func financial_statements(start_date, end_date time.Time, remove_empties bool) ([]statement, []statement) {
	start_date, end_date = check_dates(dates(start_date), dates(end_date))
	tx, _ := db.Begin()
	journal := qframe.ReadSQL(tx, qsql.Query("select date,entry_number,account,value,quantity from journal"))
	journal_before := journal.Filter(qframe.And(qframe.Filter{Column: "date", Comparator: "<", Arg: start_date.String()})).Drop("date", "entry_number", "entry_expair")
	journal_after := journal.Filter(qframe.And(qframe.Filter{Column: "date", Comparator: ">=", Arg: start_date.String()}, qframe.Filter{Column: "date", Comparator: "<=", Arg: end_date.String()})).Drop("date", "entry_number", "entry_expair")
	retained_earnings := journal_before.Filter(qframe.And(qframe.Filter{Column: "account", Comparator: "=", Arg: temporary_debit_accounts})).Drop("date", "entry_number", "entry_expair")
	fmt.Println(retained_earnings, journal_after)
	var balance_sheet, cash_flow []statement
	return balance_sheet, cash_flow
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
		log.Fatal("you order ", quantity, " but you have ", quantity-quantity_count, " ", account, " with barcode ", barcode)
	}
	return costs
}

func weighted_average(array_of_accounts []string) {
	for _, account := range array_of_accounts {
		db.Exec("update inventory set price=(select sum(value)/sum(quantity) from journal where account=?) where account=?", account, account)
	}
}

func price_discount_tax_list(account string, quantity float64) []Account_value_quantity_barcode {
	var new_array []Account_value_quantity_barcode
	for _, list := range price_discount_tax {
		if list.Account == account {
			new_array = append(new_array, Account_value_quantity_barcode{"revenue of " + account, list.Price * quantity, quantity, ""})
			new_array = append(new_array, Account_value_quantity_barcode{"discount of " + account, list.Discount * quantity, quantity, ""})
			new_array = append(new_array, Account_value_quantity_barcode{"tax of " + account, list.Tax * quantity, quantity, ""})
			new_array = append(new_array, Account_value_quantity_barcode{"tax", list.Tax * quantity, quantity, ""})
		}
	}
	return new_array
}

func entry_number() int {
	var tag int
	err = db.QueryRow("select max(entry_number) from journal").Scan(&tag)
	if err != nil {
		tag = 0
	}
	return tag + 1
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

func dates(date time.Time) time.Time {
	if date.IsZero() {
		return now
	}
	return date
}

func check_dates(start_date, end_date time.Time) (time.Time, time.Time) {
	if start_date.After(end_date) {
		log.Fatal("please enter the start_date<=end_date")
	}
	return start_date, end_date
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
		log.Fatal(duplicated_element, " is duplicated values in the fields of Financial_accounting and that make error. you should remove the duplicate")
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
		log.Fatal(duplicated_element, " is duplicated values in the fields of Financial_accounting and that make error. you should remove the duplicate")
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

func accounts_slice(args []directory_account_price_discount_tax) []string {
	accounts_slice := []string{}
	for _, i := range args {
		accounts_slice = append(accounts_slice, i.Account)
	}
	return accounts_slice
}

func accounts_name(args []directory_account) []string {
	accounts_slice := []string{}
	for _, i := range args {
		accounts_slice = append(accounts_slice, i.account_name)
	}
	return accounts_slice
}

func accounts_directory(args []directory_account_price_discount_tax) [][]uint {
	accounts_slice := [][]uint{}
	for _, i := range args {
		accounts_slice = append(accounts_slice, i.directory_no)
	}
	return accounts_slice
}

func accounts_directory1(args []directory_account) [][]uint {
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
		Company_name:               "hashem2",
		Discount:                   0,
		Invoice_discounts_tax_list: [][3]float64{{5, 0, 0}, {100, 0, 0}},
		Fifo:                       []directory_account_price_discount_tax{{[]uint{1, 3, 1}, "book1", 15, 0, 0}},
		Lifo:                       []directory_account_price_discount_tax{{[]uint{1, 3, 2}, "book2", 15, 0, 0}},
		Wma:                        []directory_account_price_discount_tax{{[]uint{1, 3, 3}, "book", 10, -1, -1}},
		Service:                    []directory_account_price_discount_tax{{[]uint{4, 3, 4}, "service revenue", 2, -1, -1}},
		Assets_normal:              []directory_account{{[]uint{1, 1}, "office equipment"}},
		Cash_and_cash_equivalent:   []directory_account{{[]uint{1, 8}, "cash"}},
		Assets_contra:              []directory_account{},
		Liabilities_normal:         []directory_account{{[]uint{2, 1}, "tax"}},
		Liabilities_contra:         []directory_account{},
		Equity_normal:              []directory_account{},
		Equity_contra:              []directory_account{},
		Withdrawals:                []directory_account{},
		Revenues:                   []directory_account{{[]uint{4, 1}, "revenue of service revenue"}, {[]uint{4, 2}, "revenue of book"}},
		Discounts:                  []directory_account{{[]uint{3, 1}, "discount of service revenue"}, {[]uint{3, 6}, "discount of book"}},
		Expenses:                   []directory_account{{[]uint{3, 4}, "tax of service revenue"}, {[]uint{3, 2}, "expair_expenses"}, {[]uint{3, 3}, "cost of book"}, {[]uint{3, 5}, "tax of book"}},
	}
	v.initialize()
	// entry, invoice, t, entry_number := journal_entry([]Account_value_quantity_barcode{{"book", 10, -10, ""}, {"cash", 90, 90, ""}}, true, 0 /*uint(entry_number())-1*/, time.Time{},
	// 	time.Time{}, "", "", "yasa", "hashem", []day_start_end{})

	// fmt.Println(invoice, t, entry_number)
	// for _, i := range entry {
	// 	fmt.Println(i)
	// }

	balance_sheet, cash_flow := financial_statements(time.Date(202, time.May, 21, 13, 00, 00, 00, time.Local), time.Date(2021, time.May, 20, 13, 00, 00, 00, time.Local), true)
	for _, i := range balance_sheet {
		fmt.Println(i)
	}
	for _, i := range cash_flow {
		fmt.Println(i)
	}
}
