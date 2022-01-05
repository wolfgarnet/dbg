
var ListFieldsEvery = 25

type PPField struct {
	Field  string
	Label  string
	Width  int
	String func(interface{}) (string, *string)
}

func (f PPField) Copy(field string) PPField {
	f.Field = field
	f.Label = field
	return f
}

func nicelyPrintString(s string, width int) string {
	if len(s) > width {
		s = s[:width]
	}
	padding := 0
	if width > len(s) {
		padding = width - len(s)
	}

	return fmt.Sprintf("%s%s", strings.Repeat(" ", padding), s)
}

func PPFieldStringPrototype(fieldName string, limit int) PPField {
	return PPField{
		Field: fieldName,
		Label: fieldName,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return nicelyPrintString("null", limit), nil
			}
			return nicelyPrintString(s.(string), limit), nil
		},
	}
}

func PPFieldStringReplacePrototype(fieldName string, limit int, label string, replace map[string]string) PPField {
	return PPField{
		Field: fieldName,
		Label: label,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return nicelyPrintString("null", limit), nil
			}
			str, ok := replace[s.(string)]
			if !ok {
				return nicelyPrintString("missing", limit), nil
			}
			return nicelyPrintString(str, limit), nil
		},
	}
}

func PPFieldFloatPrototype(fieldName string, limit int, multiplier float64) PPField {
	return PPField{
		Field: fieldName,
		Label: fieldName,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return nicelyPrintString("null", limit), nil
			}
			return fmt.Sprintf("% "+fmt.Sprintf("%d", limit)+".2f", s.(float64)*multiplier), nil
		},
	}
}

func PPFieldDatePrototype(fieldName string, limit int) PPField {
	return PPField{
		Field: fieldName,
		Label: fieldName,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return nicelyPrintString("null", limit), nil
			}
			return nicelyPrintString(s.(time.Time).Format("2006-01-02"), limit), nil
		},
	}
}

func PPFieldBoolPrototype(fieldName string, limit int) PPField {
	return PPField{
		Field: fieldName,
		Label: fieldName,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return fmt.Sprintf("%"+fmt.Sprintf("%d", limit)+"s", "null"), nil
			}
			_, isBool := s.(bool)
			if !isBool {
				return "NOT BOOL", nil
			}
			return fmt.Sprintf("%"+fmt.Sprintf("%d", limit)+"t", s.(bool)), nil
		},
	}
}

func PPFieldSlicePrototype(fieldName string, limit int, fields ...PPField) PPField {
	return PPField{
		Field: fieldName,
		Label: fieldName,
		Width: limit,
		String: func(s interface{}) (string, *string) {
			if s == nil {
				return fmt.Sprintf("%"+fmt.Sprintf("%d", limit)+"s", "none"), nil
				//return nicelyPrintString("null", limit), nil
			}
			extra := prettyPrint(s, nil, 1, 1, fields...)
			//return fmt.Sprintf("%" + fmt.Sprintf("%d", limit)), &extra
			return strings.Repeat("v", limit), &extra
		},
	}
}

var PPFieldDate = PPFieldDatePrototype("Date", 10)

var PPFieldAmountFloat = PPFieldFloatPrototype("Amount", 10, 1.0)
var PPFieldAmountDecimal = PPFieldFloatPrototype("Amount", 10, 1.0)
var PPFieldID = PPFieldStringPrototype("ID", 36)
var PPFieldSettled = PPFieldBoolPrototype("Settled", 5)

var PPFieldValidFrom = PPFieldDate.Copy("ValidFrom")
var PPFieldValidTo = PPFieldDate.Copy("ValidTo")
var PPFieldExpireAt = PPFieldDate.Copy("ExpireAt")

func PrettyPrint(data interface{}, padding int, fields ...PPField) {
	text := prettyPrint(data, nil, padding, 0, fields...)
	fmt.Println(text)
}

func PrettyPrintWithColumns(data interface{}, columns ...string) {
	PrettyPrint2(data, nil, columns...)
}

func PrettyPrint2(data interface{}, mappedReplace map[string]map[string]string, columns ...string) {
	value := reflect.ValueOf(data)
	fields := columnsInfo(value)
	selected := make([]PPField, 0)
	if len(columns) > 0 {
		for _, column := range columns {
			for _, field := range fields {
				if field.Field == column {
					selected = append(selected, field)
					break
				}
			}
		}
	} else {
		selected = fields
	}
	for i := range selected {
		replace, hasReplace := mappedReplace[selected[i].Field]
		if !hasReplace {
			continue
		}
		selected[i] = PPFieldStringReplacePrototype(selected[i].Field, selected[i].Width, selected[i].Label, replace)
	}
	text := prettyPrint(data, nil, 1, 0, selected...)
	fmt.Println(text)
}

func columnsInfo(value reflect.Value) (columns []PPField) {
	switch value.Kind() {
	case reflect.Slice:
		if value.Len() == 0 {
			return nil
		}
		return columnsInfo(value.Index(0))

	case reflect.Map:
		if value.Len() == 0 {
			return nil
		}
		for _, key := range value.MapKeys() {
			v := value.MapIndex(key)
			info := columnsInfo(v)
			if info != nil {
				return info
			}
		}

	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)

			// Find specials
			switch field.Name {
			case "ID":
				columns = append(columns, PPFieldID)
				continue
			}

			switch field.Type.String() {
			case "time.Time", "*time.Time":
				columns = append(columns, PPFieldDate.Copy(field.Name))
			case "decimal.Decimal":
				columns = append(columns, PPFieldAmountDecimal.Copy(field.Name))
			case "float64", "float32":
				columns = append(columns, PPFieldAmountFloat.Copy(field.Name))
			case "string":
				columns = append(columns, PPFieldStringPrototype(field.Name, 55))
			}
		}
	}

	return columns
}

func prettyPrint(data interface{}, title *string, padding int, depth int, fields ...PPField) string {
	if len(fields) == 0 {
		return ""
	}

	value := reflect.ValueOf(data)

	length := 1
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Map {
		length = value.Len()
	}

	if length < 1 {
		return fmt.Sprintf("Nothing to show (%s)\n", value.Type().String())
	}

	buffer := bytes.Buffer{}

	if depth == 0 {
		if title == nil {
			buffer.WriteString(fmt.Sprintf("Showing '%s'\n", value.Type().String()))
		} else {
			buffer.WriteString(fmt.Sprintf("Showing '%s'\n", *title))
		}
	}

	margin := strings.Repeat(" ", depth*4)
	spacing := 0
	for length != 0 {
		length /= 10
		spacing += 1
	}

	displayFields := func() {
		// Print fields
		buffer.WriteString(margin)
		for i, field := range fields {
			if i > 0 {
				buffer.WriteString(strings.Repeat(" ", padding))
			} else {
				buffer.WriteString(fmt.Sprintf(strings.Repeat(" ", spacing+3)))
			}
			max := len(field.Label)
			filler := ""
			if field.Width < max {
				max = field.Width
			} else {
				filler = strings.Repeat("-", field.Width-max)
			}
			buffer.WriteString(fmt.Sprintf("%s%s", filler, field.Label[:max]))
		}
		buffer.WriteString("\n")
	}

	switch value.Kind() {
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if i%ListFieldsEvery == 0 {
				displayFields()
			}
			buffer.WriteString(margin)
			buffer.WriteString(PrettyPrintStruct(reflect.ValueOf(i), value.Index(i), i, value.Len(), spacing, padding, fields...))
		}

	case reflect.Map:
		displayFields()
		for i, key := range value.MapKeys() {
			buffer.WriteString(PrettyPrintStruct(key, value.MapIndex(key), i, value.Len(), spacing, padding, fields...))
		}

	case reflect.Struct:
		displayFields()
		buffer.WriteString(PrettyPrintStruct(reflect.ValueOf(""), value, 0, 1, spacing, padding, fields...))

	default:
		buffer.WriteString(fmt.Sprintf("WHAT %v?", value.Type().Kind()))
	}

	return buffer.String()
}

func PrettyPrintStruct(key reflect.Value, value reflect.Value, idx int, total int, spacing int, padding int, fields ...PPField) string {
	buffer := bytes.Buffer{}
	switch value.Kind() {
	case reflect.Struct:
		var extras []string
		for i, field := range fields {
			if i > 0 {
				buffer.WriteString(strings.Repeat(" ", padding))
			} else {
				if total > 0 {
					buffer.WriteString(fmt.Sprintf("[%"+fmt.Sprintf("%d", spacing)+"d] ", idx))
				} else {
					buffer.WriteString(" ")
				}
			}
			structField := value.FieldByName(field.Field)
			s, extra := PrettyPrintValue(structField, field)
			buffer.WriteString(s)
			if extra != nil && len(*extra) > 0 {
				extras = append(extras, *extra)
			}
		}
		for _, e := range extras {
			buffer.WriteByte('\n')
			buffer.WriteString(e)
		}
		buffer.WriteString("\n")

	case reflect.Slice:
		title := key.String()
		return prettyPrint(value.Interface(), &title, 1, 0, fields...)

	default:
		buffer.WriteString(fmt.Sprintf("Not a struct! %v", value.Kind()))
	}

	return buffer.String()
}

func PrettyPrintValue(value reflect.Value, field PPField) (string, *string) {
	switch value.Kind() {

	case reflect.Bool:
		return field.String(value.Bool())

	case reflect.Ptr:
		if value.IsNil() {
			return field.String(nil)
		}

		value = value.Elem()
		fallthrough

	case reflect.Struct:
		switch value.Type().Name() {
		case "Time":
			return field.String(value.Interface())
		case "Decimal":
			d, ok := value.Interface().(decimal.Decimal)
			if !ok {
				return "Unsupported decimal!", nil
			}
			float, _ := d.Float64()
			return field.String(float)
		default:
			return "Unsupported struct!", nil
		}

	case reflect.Float64:
		return field.String(value.Float())

	case reflect.String:
		return field.String(value.String())

	case reflect.Slice:
		return field.String(value.Interface())

	default:
		if value.IsNil() {
			return "NIL", nil
		} else {
			return fmt.Sprintf("EH? %v", value.Kind()), nil
		}
	}
}
