import pandas as pd

# Load the Excel file, using row 2 (0-based index) as header
file_path = '1 KK_ART Pondokrejo.xlsx'
df = pd.read_excel(file_path, header=2)

# Print columns with their indices
for i, col in enumerate(df.columns):
    print(f"{i}: {col}")
