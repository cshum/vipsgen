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

GObjectClass* get_object_class(void* obj);

// Function to get operation arguments structure
typedef struct {
    char *name;
    char *nick;
    char *blurb;
    int flags;
    GType type_val;
    int is_input;
    int is_output;
    int required;
} ArgInfo;

// Get all arguments of an operation
ArgInfo* get_operation_arguments(const char *operation_name, int *count);

// Free operation arguments
void free_operation_arguments(ArgInfo *args, int count);

// Helper functions for type checking
int is_type_enum(GType type);
int is_type_flags(GType type);
char* get_type_name(GType type);
