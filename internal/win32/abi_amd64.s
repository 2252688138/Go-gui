#include "textflag.h"

// 说明：Win64 上带 float 参数的 COM 调用已改为运行时生成机器码（com_float_amd64.go），
// 手写汇编桥因 ABI0/ABIInternal 转换与 SP 对齐问题实测不可靠，这里不再保留实现。
