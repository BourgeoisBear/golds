package comments

func foo() {
	if a := 1 == 1; a {
		/* ssssss */
	} else {
		// dafafasfa
	}
	if a := 1 == 1; a {
		/* ssssss */
	} else /*da faadsfaf /**/ /* /*dafafaf*/ // dafasfasfas
	//dafafasfasfa /*daafafaf/*
	{
		// dafafasfa
	}

	var x = []int{}
	for range x {
	}

	for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
	}
	for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
	}
	for range //aaa
	//aaaa
	x {
	}

	for _, _ = range x {
	}

	for _, _ = range /*da faadsfaf /**/ /* /*dafafaf*/ x {

		for range //aaa
		//aaaa
		x {
			for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
				for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
					if a := 1 == 1; a {
						/* ssssss */
					} else {
						// dafafasfa
					}
					if a := 1 == 1; a {
						/* ssssss */
					} else /*da faadsfaf /**/ /* /*dafafaf*/ // dafasfasfas
					//dafafasfasfa /*daafafaf/*
					{
						// dafafasfa
					}
				}
			}
		}

		for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
		}
		for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
		}
	}

	for a, _ := range x {
		_ = a
	}
}

func foo2() {
	if true {
	}/*aaa*//*bbbb else*/else {
	}

	for /*aaa*///bbb
/*ccc*//*dddd*/
//eeee	range
	range [3]int{} {}
	
	switch func() interface{} {
		return interface{}(1)
	}().(/*aaa*//*bbbb*///cccc
//ddd type
/*eee

*///fff type
		type/*ggg*/ ) {
	}
}

/*dddd*//*dddddd
*/// aaaaa

