import pandas as pd

# Load the Excel file, reading more rows to find the header
file_path = '1 KK_ART Pondokrejo.xlsx'
df = pd.read_excel(file_path, nrows=10, header=None)

# Print the first few rows
print(df.head(10))
