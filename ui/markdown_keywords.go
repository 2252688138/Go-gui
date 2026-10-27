package ui

// 语法着色关键字表（按语言分类）。放到独立文件，避免与扫描器逻辑混在一起。

var mdGoKeywords = mdSet("break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var")
var mdGoTypes = mdSet("bool byte complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr any")
var mdGoBuiltins = mdSet("append cap close complex copy delete imag len make new panic print println real recover nil true false iota")

var mdJSKeywords = mdSet("async await break case catch class const continue debugger default delete do else export extends finally for from function get if import in instanceof let new of return set static super switch this throw try typeof var void while with yield interface type enum implements declare namespace readonly public private protected as")
var mdJSTypes = mdSet("any boolean number string symbol unknown never object bigint void Array Promise Map Set Record Partial Required Readonly")
var mdJSBuiltins = mdSet("console window document Math JSON Date Promise Array Object String Number Boolean RegExp Error require module process globalThis undefined null true false NaN")

var mdJavaKeywords = mdSet("abstract as assert break case catch class const continue default do else enum extends final finally for fun goto if implements import instanceof interface internal native new null open operator override package private protected public return sealed static strictfp super switch synchronized this throw throws transient try val var volatile when while yield record")
var mdJavaTypes = mdSet("boolean byte char double float int long short void String Integer Long Double Float Character Boolean List Map Set HashMap ArrayList Object Any Unit")

var mdCKeywords = mdSet("auto break case const continue default do else enum extern for goto if inline register restrict return sizeof static struct switch typedef union volatile while class namespace template public private protected virtual override new delete nullptr this using try catch throw constexpr noexcept operator friend mutable explicit")
var mdCTypes = mdSet("bool char double float int long short signed unsigned void size_t uint8_t uint16_t uint32_t uint64_t int8_t int16_t int32_t int64_t wchar_t")

var mdPyKeywords = mdSet("and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield match case self True False None")
var mdPyBuiltins = mdSet("print len range str int float list dict set tuple bool bytes open type isinstance super enumerate zip map filter sum min max abs sorted")

var mdSQLKeywords = mdSet("select from where insert into values update delete create table alter drop index join inner left right outer full on group by order having limit offset union all distinct as and or not null is in between like exists case when then else end primary key foreign references default constraint unique view database begin commit rollback transaction")

var mdShellKeywords = mdSet("if then else elif fi for while do done case esac function return in export local readonly declare unset shift source alias echo cd pwd ls cat grep sed awk cp mv rm mkdir rmdir touch chmod chown sudo curl wget git go npm yarn pnpm node python pip docker kubectl powershell param Write-Host Get-ChildItem")

var mdCSSKeywords = mdSet("important inherit initial unset auto none flex grid block inline absolute relative fixed sticky hidden visible")
