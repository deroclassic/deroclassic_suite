/* Factorial implementation in DVM-BASIC   */


/* iterative implementation of factorial */
         Function Factorial(input Uint64) Uint64
	10  dim result,input_copy as Uint64
	15  LET input_copy =  input
	20  LET result = 1
	30  LET result = result * input
	40  LET input = input - 1
	50  IF input >= 2 THEN GOTO 30
	60  printf "FACTORIAL of %d = %d" input_copy result 
	70  RETURN result
	End Function

/* recursive implementation of factorial */
	Function Factorial_recursive(input Uint64) Uint64
	10  IF input == 1 THEN GOTO 20 ELSE GOTO 30
	20  RETURN 1
	30  RETURN  input * Factorial_recursive(input - 1)
	End Function
	
	Function Factorialr(input Uint64) Uint64
	10 dim result as Uint64
	20 LET result = Factorial_recursive(input)
        30 printf "FACTORIAL of %d = %d " input result  
	40 RETURN result
	End Function
	


