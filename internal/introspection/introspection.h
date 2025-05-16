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
    int has_default;
    int default_type;  // 1=bool, 2=int, 3=double, 4=string

    // Specific default values for each type
    gboolean bool_default;
    gint int_default;
    gdouble double_default;
    char *string_default;

    // Additional type information
    int is_image;      // Is this an image parameter?
    int is_buffer;     // Is this a buffer parameter?
    int is_array;      // Is this an array parameter?
} ArgInfo;

// Get all arguments of an operation
ArgInfo* get_operation_arguments(const char *operation_name, int *count);

// Free operation arguments
void free_operation_arguments(ArgInfo *args, int count);

// Helper functions for type checking
int is_type_enum(GType type);
int is_type_flags(GType type);
char* get_type_name(GType type);

// Operation information structure
typedef struct {
    char *name;
    char *nickname;
    char *description;
    int flags;
} OperationInfo;

// Get all available operations
OperationInfo* get_all_operations(int *count);

// Free operation information
void free_operation_info(OperationInfo *ops, int count);

// Get operation details including whether it has image input/output
typedef struct {
    int has_image_input;
    int has_image_output;
    int has_one_image_output;
    int has_buffer_input;
    int has_buffer_output;
    int has_array_image_input;
    char *category;
} OperationDetails;

// Get detailed information about an operation
OperationDetails get_operation_details(const char *operation_name);
