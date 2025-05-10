#include <vips/vips.h>
#include <stdlib.h>

// Data structures
typedef struct {
    char **names;
    int count;
    int capacity;
} OperationList;

typedef struct {
    char *name;
    int value;
    char *nick;
} EnumValueInfo;

char** get_all_operation_names(int *count);

void free_operation_names(char **names, int count);

EnumValueInfo* get_enum_values(const char *enum_type_name, int *count);

int type_exists(const char *type_name);

void free_enum_values(EnumValueInfo *values, int count);
